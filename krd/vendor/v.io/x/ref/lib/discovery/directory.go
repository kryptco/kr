// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package discovery

import (
	"sync"

	"v.io/v23"
	"v.io/v23/context"
	"v.io/v23/discovery"
	"v.io/v23/naming"
	"v.io/v23/options"
	"v.io/v23/rpc"
	"v.io/v23/security"
)

const maxTotalAttachmentSize = 4096

type dirServer struct {
	d *idiscovery

	mu       sync.Mutex
	adMap    map[discovery.AdId]*AdInfo // GUARDED_BY(mu)
	dirAddrs []string                   // GUARDED_BY(mu)

	cancel func()
}

func (s *dirServer) Lookup(ctx *context.T, _ rpc.ServerCall, id discovery.AdId) (AdInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	adinfo := s.adMap[id]
	if adinfo == nil {
		return AdInfo{}, NewErrAdvertisementNotFound(ctx, id)
	}

	copied := *adinfo
	copied.Ad.Attachments = make(discovery.Attachments)

	sizeSum := 0
	for k, v := range adinfo.Ad.Attachments {
		if sizeSum+len(v) > maxTotalAttachmentSize {
			copied.Status = AdPartiallyReady
			continue
		}
		sizeSum += len(v)
		copied.Ad.Attachments[k] = v
	}
	return copied, nil
}

func (s *dirServer) GetAttachment(ctx *context.T, _ rpc.ServerCall, id discovery.AdId, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	adinfo := s.adMap[id]
	if adinfo == nil {
		return nil, NewErrAdvertisementNotFound(ctx, id)
	}
	return adinfo.Ad.Attachments[name], nil
}

func (s *dirServer) publish(adinfo *AdInfo) {
	s.mu.Lock()
	adinfo.DirAddrs = s.dirAddrs
	s.adMap[adinfo.Ad.Id] = adinfo
	s.mu.Unlock()
}

func (s *dirServer) unpublish(id discovery.AdId) {
	s.mu.Lock()
	delete(s.adMap, id)
	s.mu.Unlock()
}

func (s *dirServer) shutdown() {
	s.cancel()
}

func (s *dirServer) updateDirAddrs(addrs []string) {
	s.mu.Lock()
	s.dirAddrs = addrs
	newAdinfos := make([]*AdInfo, 0, len(s.adMap))
	for id, adinfo := range s.adMap {
		newAdinfo := *adinfo
		newAdinfo.DirAddrs = s.dirAddrs
		s.adMap[id] = &newAdinfo
		newAdinfos = append(newAdinfos, &newAdinfo)
	}
	s.mu.Unlock()

	for _, adinfo := range newAdinfos {
		s.d.updateAdvertising(adinfo)
	}
}

func newDirServer(ctx *context.T, d *idiscovery, opts ...rpc.ServerOpt) (*dirServer, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &dirServer{d: d, adMap: make(map[discovery.AdId]*AdInfo), cancel: cancel}
	ctx, server, err := v23.WithNewServer(ctx, "", DirectoryServer(s), security.AllowEveryone(), opts...)
	if err != nil {
		cancel()
		return nil, err
	}

	status := server.Status()
	curAddrs := sortedNames(status.Endpoints)
	s.updateDirAddrs(curAddrs)

	go func() {
		for {
			select {
			case <-status.Dirty:
				status = server.Status()
				if status.State != rpc.ServerActive {
					return
				}
				newAddrs := sortedNames(status.Endpoints)
				if sortedStringsEqual(curAddrs, newAddrs) {
					continue
				}

				s.updateDirAddrs(newAddrs)
				curAddrs = newAddrs
			case <-ctx.Done():
				return
			}
		}
	}()
	return s, nil
}

type dirClient struct {
	dirAddrs []string
}

func (c *dirClient) Lookup(ctx *context.T, id discovery.AdId) (*AdInfo, error) {
	resolved, err := c.resolveDirAddrs(ctx)
	if err != nil {
		return nil, err
	}
	adinfo, err := DirectoryClient("").Lookup(ctx, id, resolved)
	if err != nil {
		return nil, err
	}
	return &adinfo, nil
}

func (c *dirClient) GetAttachment(ctx *context.T, id discovery.AdId, name string) ([]byte, error) {
	resolved, err := c.resolveDirAddrs(ctx)
	if err != nil {
		return nil, err
	}
	return DirectoryClient("").GetAttachment(ctx, id, name, resolved)
}

func (c *dirClient) resolveDirAddrs(ctx *context.T) (options.Preresolved, error) {
	// We pre-resolve the addresses so that we can utilize the RPC system's handling
	// of connecting to the same server by trying multiple endpoints.
	ns := v23.GetNamespace(ctx)
	resolved := naming.MountEntry{IsLeaf: true}
	for _, addr := range c.dirAddrs {
		entry, err := ns.Resolve(ctx, addr)
		if err != nil {
			return options.Preresolved{}, err
		}
		resolved.Servers = append(resolved.Servers, entry.Servers...)
	}
	return options.Preresolved{&resolved}, nil
}

func newDirClient(dirAddrs []string) *dirClient {
	return &dirClient{dirAddrs}
}
