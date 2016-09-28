// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import "CBAdvertisingDriver.h"
#import "CBDriver.h"
#import "CBLog.h"
#import "CBScanningDriver.h"
#import "CBUtil.h"

static CBDriver *_instance = nil;

#define kCBDriverQueueName "io.v.v23.CoreBluetoothDriver"
NSString *kCBDriverErrorDomain = @"io.v.v23.CoreBluetoothDriver";

@interface CBDriver ()
@end

@implementation CBDriver

+ (CBDriver *_Nonnull)instance {
  @synchronized(self) {
    if (!_instance) {
      _instance = [CBDriver new];
    }
  }
  return _instance;
}

- (id)init {
  if (self = [super init]) {
    // Create the serial dispatch queue for the driver
    self.queue = dispatch_queue_create(kCBDriverQueueName, NULL);
    self.advertisingDriver = [[CBAdvertisingDriver alloc] initWithQueue:self.queue];
    self.scanningDriver = [[CBScanningDriver alloc] initWithQueue:self.queue];
  }
  return self;
}

+ (void)shutdown {
  @synchronized(self) {
    if (!_instance) return;
    CBDispatchSync(_instance.queue, ^{
      _instance.scanningDriver = nil;
      _instance.advertisingDriver = nil;
    });
    _instance = nil;
  }
}

- (NSString *)debugDescription {
  return [NSString stringWithFormat:@"[CBDriver\nadvertising=%@\ndiscovery=%@]",
                                    self.scanningDriver, self.advertisingDriver];
}

@end

static void copyString(NSString *string, char **dst);

char *v23_cbdriver_debug_string() {
  char *dst = NULL;
  copyString([[CBDriver instance] debugDescription], &dst);
  return dst;
}

BOOL v23_cbdriver_addService(const char *_Nonnull cUuid,
                             CBDriverCharacteristicMapEntry *_Nonnull entries, int entriesLength,
                             char *_Nullable *_Nullable errorOut) {
  CBUUID *uuid = [CBUUID UUIDWithString:[NSString stringWithUTF8String:cUuid]];
  NSMutableDictionary<CBUUID *, NSData *> *_Nonnull characteristics = [NSMutableDictionary new];
  for (int i = 0; i < entriesLength; i++) {
    CBDriverCharacteristicMapEntry entry = entries[i];
    CBUUID *characteristicUuid = [CBUUID UUIDWithString:[NSString stringWithUTF8String:entry.uuid]];
    NSData *data = [[NSData alloc] initWithBytes:entry.data length:entry.dataLength];
    characteristics[characteristicUuid] = data;
  }
  // We're on a go thread -- addService in obj-c will run on the bluetooth queue and callback
  // from that queue. Thus we are able to block until we get the response (or timeout).
  dispatch_semaphore_t condition = dispatch_semaphore_create(0);
  __block NSString *err = nil;
  CBInfoLog(@"Adding service %@ with characteristics %@", uuid, characteristics);
  [CBDriver.instance.advertisingDriver
           addService:uuid
      characteristics:characteristics
             callback:^(CBUUID *_Nonnull callbackUuid, NSError *_Nullable error) {
               assert([uuid isEqual:callbackUuid]);
               CBDebugLog(@"Got callback on add service %@ with error %@", callbackUuid, error);
               if (error) {
                 err = [NSString stringWithFormat:@"%@", error];
               }
               dispatch_semaphore_signal(condition);
             }];
  // Wait up to 5 seconds for the callback
  dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(5 * NSEC_PER_SEC));
  if (dispatch_semaphore_wait(condition, timeout) && !err) {
    err = @"Timeout adding service -- CoreBluetooth did not return in time";
    CBInfoLog(err);
  }
  if (err) {
    copyString(err, errorOut);
    return NO;
  }
  return YES;
}

int v23_cbdriver_advertisingServiceCount() {
  return (int)[CBDriver instance].advertisingDriver.serviceCount;
}

BOOL v23_cbdriver_writeData(const char *_Nonnull data,
                             const int dataLength, char *_Nullable *_Nullable errorOut) {
	NSData* nsData = [NSData dataWithBytes:data length:dataLength];
  // We're on a go thread -- writeData in obj-c will run on the bluetooth queue and callback
  // from that queue. Thus we are able to block until we get the response (or timeout).
  dispatch_semaphore_t condition = dispatch_semaphore_create(0);
  __block NSString *err = nil;
  CBInfoLog(@"Writing data to characteristic");
  [CBDriver.instance.advertisingDriver
           writeData:nsData
             callback:^(BOOL result, NSError *_Nullable error) {
               CBDebugLog(@"Got callback on writeData %@ with error %@", result, error);
               if (error) {
                 err = [NSString stringWithFormat:@"%@", error];
               }
               dispatch_semaphore_signal(condition);
             }];
  // Wait up to 5 seconds for the callback
  dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, (int64_t)(5 * NSEC_PER_SEC));
  if (dispatch_semaphore_wait(condition, timeout) && !err) {
    err = @"Timeout writing data -- CoreBluetooth did not return in time";
    CBInfoLog(err);
  }
  if (err) {
    copyString(err, errorOut);
    return NO;
  }
  return YES;
}

static CBDriverCharacteristicMapEntry *characteristicsMapToEntries(
    NSDictionary<CBUUID *, NSData *> *_Nullable characteristics);

static void freeCharacteristicsEntriesMap(CBDriverCharacteristicMapEntry *entries, int length);

// This is exported from go
extern void v23_corebluetooth_scan_handler_on_discovered(
    const char *_Nonnull uuid, CBDriverCharacteristicMapEntry *_Nonnull entries, int entriesLength,
    int rssi);

void v23_cbdriver_removeService(const char *_Nonnull cUuid) {
  CBUUID *uuid = [CBUUID UUIDWithString:[NSString stringWithUTF8String:cUuid]];
  [[CBDriver instance].advertisingDriver removeService:uuid];
}

void v23_cbdriver_setAdRotateDelay(float seconds) {
  [CBDriver instance].advertisingDriver.rotateAdDelay = (NSTimeInterval)seconds;
}

BOOL v23_cbdriver_startScan(const char *_Nonnull *_Nonnull cUuids, int uuidsLength,
                            const char *_Nonnull cBaseUuid, const char *_Nonnull cMaskUuid,
                            char *_Nullable *_Nullable errorOut) {
  NSMutableArray<CBUUID *> *uuids = [NSMutableArray new];
  for (int i = 0; i < uuidsLength; i++) {
    [uuids addObject:[CBUUID UUIDWithString:[NSString stringWithUTF8String:cUuids[i]]]];
  }
  CBUUID *baseUuid = [CBUUID UUIDWithString:[NSString stringWithUTF8String:cBaseUuid]];
  CBUUID *maskUuid = [CBUUID UUIDWithString:[NSString stringWithUTF8String:cMaskUuid]];
  NSError *err = nil;
  BOOL success = [CBDriver.instance.scanningDriver
      startScan:uuids
       baseUuid:baseUuid
       maskUuid:maskUuid
        handler:^(CBUUID *_Nonnull uuid,
                  NSDictionary<CBUUID *, NSData *> *_Nullable characteristics, int rssi) {
          // Go assumes characteristics -- if empty then it crashes.
          if (characteristics.count == 0) {
            CBErrorLog(@"Got vanadium service %@ but no characteristics; ignoring", uuid);
            return;
          }
          // Call go
          CBDriverCharacteristicMapEntry *entries = characteristicsMapToEntries(characteristics);
          v23_corebluetooth_scan_handler_on_discovered(uuid.UUIDString.UTF8String, entries,
                                                       (int)characteristics.count, rssi);
          // Clean up
          freeCharacteristicsEntriesMap(entries, (int)characteristics.count);
        }
          error:&err];
  if (!success || err) {
    copyString([NSString stringWithFormat:@"%@", err], errorOut);
  }
  return success;
}

void v23_cbdriver_stopScan() { [[CBDriver instance].scanningDriver stopScan]; }

void v23_cbdriver_clean() { [CBDriver shutdown]; }

static CBDriverCharacteristicMapEntry *characteristicsMapToEntries(
    NSDictionary<CBUUID *, NSData *> *_Nullable characteristics) {
  if (!characteristics.count) return NULL;
  CBDriverCharacteristicMapEntry *entries =
      malloc(sizeof(CBDriverCharacteristicMapEntry) * characteristics.count);
  if (!entries) {
    return NULL;
  }
  int i = 0;
  for (CBUUID *uuid in characteristics) {
    CBDriverCharacteristicMapEntry entry;
    NSData *data = characteristics[uuid];
    copyString(uuid.UUIDString, &entry.uuid);
    entry.data = data.bytes;
    entry.dataLength = (int)data.length;
    entries[i] = entry;
    i++;
  }
  return entries;
}

static void freeCharacteristicsEntriesMap(CBDriverCharacteristicMapEntry *entries, int length) {
  if (!entries) {
    return;
  }
  for (int i = 0; i < length; i++) {
    CBDriverCharacteristicMapEntry entry = entries[i];
    free(entry.uuid);
    // The data will free itself as it was never copied
  }
  free(entries);
}

static void copyString(NSString *string, char **dst) {
  if (!dst) {
    CBErrorLog(@"Missing dst string, not copying %@", string);
    return;
  }
  NSUInteger length = [string lengthOfBytesUsingEncoding:NSUTF8StringEncoding] + sizeof("\0");
  *dst = malloc(length);
  if (*dst) {
    [string getCString:*dst maxLength:length encoding:NSUTF8StringEncoding];
  }
}

@implementation CBUUID (Description)

- (NSString *)description {
  return [NSString stringWithFormat:@"[CBUUID %@]", self.UUIDString];
}

@end
