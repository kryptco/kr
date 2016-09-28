// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <CoreBluetooth/CoreBluetooth.h>

/** CBOnDiscoveredHandler is called when a target Vanadium service has been discovered.

 Optionally the received signal strength in dBm can be passed to rssi.
 The valid range is [-127, 0).
 */
typedef void (^CBOnDiscoveredHandler)(CBUUID *_Nonnull uuid,
                                      NSDictionary<CBUUID *, NSData *> *_Nullable characteristics,
                                      int rssi);

@interface CBScanningDriver : NSObject<CBCentralManagerDelegate>

@property(nonatomic, strong) dispatch_queue_t _Nonnull queue;
@property(nonatomic, strong) CBCentralManager *_Nonnull central;

- (id _Nullable)initWithQueue:(dispatch_queue_t _Nonnull)queue;

/** StartScan starts BLE scanning for the specified uuids and the scan results will be
 delivered through the scan handler.

 An empty uuids means all Vanadium services. The driver may use baseUuid and maskUuid
 to filter Vanadium services.

 It is guaranted that there is at most one active scan at any given time. That is,
 StopScan() will be called before starting a new scan.

 This method will start and run regardless if the BLE hardware is turned on or off. That is to say,
 it will remember the scan query in between power downs of the BLE hardware such that when it comes
 back up the scan will resume automatically. Do not rely on this method returning an error if the
 hardware is off, because it won't.

 @param uuids Service uuids to explicitly scan for (optimizes battery if passed). Maybe an empty
 list, which would then return all Vanadium services as specified by the baseUuid and maskUuid.

 @param baseUuid The base uuid shared by all Vandium services.

 @param maskUUID The mask to apply to an arbitrary discovered uuid to see if it's equivalent to
                 baseUuid

 @param handler Handler is called on the driver's dispatch queue when it's discovered a new device
                that matches the scan's parameters.

 @param error Return mechanism for any errors in starting the scan immediately. If non-nil, then
              the scan has failed and there is no need to call stopScan -- currently this only
              occurs if the hardware is unsupported (such as running in an iOS simulator).
 */
- (BOOL)startScan:(NSArray<CBUUID *> *_Nonnull)uuids
         baseUuid:(CBUUID *_Nonnull)baseUuid
         maskUuid:(CBUUID *_Nonnull)maskUuid
          handler:(CBOnDiscoveredHandler _Nonnull)handler
            error:(NSError *_Nullable *_Nullable)error;

/** StopScan stops BLE scanning. */
- (void)stopScan;

@end
