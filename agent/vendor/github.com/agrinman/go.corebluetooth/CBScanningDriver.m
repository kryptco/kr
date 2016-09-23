// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import "CBDriver.h"
#import "CBLog.h"
#import "CBScanningDriver.h"
#import "CBUtil.h"

// These 3 constants are from v.io/x/ref/lib/disocvery/plugins/ble/encoding.go
static NSString *kPackedCharacteristicUuidFmt = @"31ca10d5-0195-54fa-9344-25fcd7072e%x%x";
static const int kMaxNumPackedServices = 16;
static const int kMaxNumPackedCharacteristicsPerService = 16;

static const NSTimeInterval kFlushCachePeriod = 24 * 60 * 60;  // Every 24h
static const NSTimeInterval kMaxTimeForDiscoverServices = 90;

@class CBPeripheralScan, CBPeripheralServiceScan;

/**
 Scanning has two major components to it: waiting for the hardware to discover service uuids,
 and connecting to a device to retrieve all advertisement services & their data (by reading
 its charactertistics). Because all BLE calls are asynchronous and CoreBluetooth requires us to
 maintain strong references to the CBPeripheral, classes CBPeripheralScan and CBPeripheralSeviceScan
 were created to encapsulate and store all state.

 CBPeripheralScan controls scanning a discovered foreign BLE device

 CBPeripheralServiceScan controls connecting to a service and reading its characteristics.

 Each of these classes performs callbacks to their delegate when they have completed a task. The
 flow goes upwards, from CBPeripheralServiceScan to its parent CBPeripheralScan, which then
 reports any discovered service to CBScanningDriver which finally reports to the passed handler.

 The following pseduo-code encapsulates the main flow (in reality everything is async):

 // CBScanningDriver
 onPeripheralDiscovered(peripheral):
   if peripheral.visibleServiceUUIDs.filter(requestedServiceUuids, baseUuid, baseMask).isEmpty:
      return
   scan = CBPeripheralScan(peripheral, requestedServiceUuids, baseUuid, baseMask)
   scan.start()

 // CBPeripheralScan.start
 allServices = peripheral.getAllServiceUuids()
 v23Services = allServices.filter(requestedServiceUuids, baseUuid, baseMask)
 foreach service in v23Service:
   serviceScan = CBPeripheralServiceScan(service)
   serviceScan.start()

 // CBPeripheralServiceScan.start
 NSMutableDictionary data = service.allCharacteristics.map { characteristic -> characteristic.read()
 }
 onDiscoveredHandler(service, data) // Here is where we callback to the original requester,
 // although in reality this is bubbled all the way back up the chain to CBScanningDriver who
 // is the one who owns and calls the handler.
 */

/** Callbacks from CBPeripheralScan */
@protocol CBPeripheralScanDelegate<NSObject>
- (void)peripheralScanDidComplete:(CBPeripheralScan *)peripheralScan;
// Returns true if a given serviceUuid matches the current scan query.
- (BOOL)uuidMatchesScanFilter:(CBUUID *)serviceUuid;
@end

/** Callbacks from CBPeripheralServiceScan */
@protocol CBPeripheralServiceScanDelegate<NSObject>
- (void)serviceScanDidComplete:(CBPeripheralServiceScan *)serviceScan;
@optional
- (void)updateLastActivity;
@end

/** CBPeripheralScan controls scanning a discovered foreign BLE device. */
@interface CBPeripheralScan : NSObject<CBPeripheralDelegate, CBPeripheralServiceScanDelegate>
@property(nonatomic, weak) id<CBPeripheralScanDelegate, CBPeripheralServiceScanDelegate> delegate;
@property(nonatomic, strong) NSDate *_Nonnull lastActivity;
@property(nonatomic, strong) CBPeripheral *_Nonnull peripheral;
@property(nonatomic, strong) NSNumber *_Nonnull rssi;
@property(nonatomic, strong) NSMutableSet<CBPeripheralServiceScan *> *_Nonnull serviceScans;
@property(nonatomic, strong) NSMutableSet<CBPeripheralServiceScan *> *_Nonnull completedScans;
- (id)initWithPeripheral:(CBPeripheral *_Nonnull)peripheral
                    rssi:(NSNumber *)rssi
                delegate:(id<CBPeripheralScanDelegate, CBPeripheralServiceScanDelegate>)delegate;
- (void)start:(NSArray<CBUUID *> *_Nullable)scanUuids;
- (NSMutableSet<CBUUID *> *_Nonnull)seenUUIDs;
@end

/** CBPeripheralServiceScan controls connecting to a service and reading its characteristics. */
@interface CBPeripheralServiceScan : NSObject<CBPeripheralDelegate>
@property(nonatomic, weak) id<CBPeripheralServiceScanDelegate> delegate;
@property(nonatomic, strong) CBService *_Nonnull service;
@property(nonatomic, strong) NSMutableSet<CBCharacteristic *> *_Nonnull queryingCharacteristics;
@property(nonatomic, strong) NSMutableDictionary<CBUUID *, NSData *> *_Nonnull characteristics;
@property(nonatomic, strong) NSNumber *_Nonnull rssi;
- (id)initWithService:(CBService *_Nonnull)service
                 rssi:(NSNumber *)rssi
             delegate:(id<CBPeripheralServiceScanDelegate>)delegate;
- (void)start;
@end

@interface CBScanningDriver ()<CBPeripheralScanDelegate, CBPeripheralServiceScanDelegate>
@property(nonatomic, assign) BOOL isScanning;
@property(nonatomic, strong) NSArray<CBUUID *> *_Nullable scanUuids;
@property(nonatomic, strong) CBUUID *_Nullable baseUuid;
@property(nonatomic, strong) CBUUID *_Nullable maskUuid;
@property(nonatomic, strong) CBOnDiscoveredHandler _Nullable onDiscoveredHandler;
@property(nonatomic, strong) NSMutableSet<CBPeripheralScan *> *_Nonnull scans;
@end

@implementation CBScanningDriver

- (id _Nullable)initWithQueue:(dispatch_queue_t _Nonnull)queue {
  if (self = [super init]) {
    self.queue = queue;
    self.central =
        [[CBCentralManager alloc] initWithDelegate:self
                                             queue:self.queue
                                           options:@{
#if TARGET_OS_IPHONE
                                             CBCentralManagerOptionShowPowerAlertKey : @NO
#endif
                                           }];
    [self _initDiscovery];
  }
  return self;
}

- (void)dealloc {
  CBDispatchSync(self.queue, ^{
    self.central.delegate = nil;
    if ([self isHardwarePoweredOn]) {
      [self.central stopScan];
    }
  });
}

/** Schedules flushing the scan cache using the default scan cache period which is currently
 one day. The reason is that we want to avoid re-connecting to services unless their advertised
 UUIDs change since updated discovery information results in a bit-flip in existing advertised
 UUIDs. To prevent a situation where we have a buggy situation that never otherwise resolves,
 we create a maximum time of 24h to hold onto our cache of devices & seen service UUIDs.
 */
- (void)scheduleFlush {
  dispatch_time_t flushCacheDelay =
      dispatch_time(DISPATCH_TIME_NOW, (int64_t)(kFlushCachePeriod * NSEC_PER_SEC));
  __weak typeof(self) this = self;
  dispatch_after(flushCacheDelay, self.queue, ^{
    [this flushScanCache];
  });
}

/** Removes all previously seen devices & service UUIDs from our cache. If we were in the middle
 of a scan, that data is preserved and the scan is restarted
 */
- (void)flushScanCache {
  NSArray *scanUuids = self.scanUuids;
  CBUUID *baseUuid = self.baseUuid;
  CBUUID *maskUuid = self.maskUuid;
  CBOnDiscoveredHandler onDiscoveredHandler = self.onDiscoveredHandler;
  BOOL wasScanning = self.isScanning;
  if (wasScanning) {
    [self stopScan];
  }
  [self _initDiscovery];
  if (wasScanning) {
    NSError *err = nil;
    [self startScan:scanUuids
           baseUuid:baseUuid
           maskUuid:maskUuid
            handler:onDiscoveredHandler
              error:&err];
    if (err) {
      CBErrorLog(@"Unable to restart scan post-flush: %@", err);
    }
  }
}

- (void)_initDiscovery {
  self.isScanning = NO;
  self.scanUuids = nil;
  self.baseUuid = nil;
  self.maskUuid = nil;
  self.onDiscoveredHandler = nil;
  self.scans = [NSMutableSet new];
  [self scheduleFlush];
}

- (BOOL)startScan:(NSArray<CBUUID *> *_Nonnull)uuids
         baseUuid:(CBUUID *_Nonnull)baseUuid
         maskUuid:(CBUUID *_Nonnull)maskUuid
          handler:(CBOnDiscoveredHandler _Nonnull)handler
            error:(NSError *_Nullable *_Nullable)error {
  __block NSError *localError = nil;
  // CoreBluetooth prefers nil to signal scan all devices
  if (uuids.count == 0) {
    uuids = nil;
  }
  CBDispatchSync(self.queue, ^{
    switch (self.central.state) {
      case CBCentralManagerStateUnsupported:
        localError = [NSError errorWithDomain:kCBDriverErrorDomain
                                         code:CBDriverErrorUnsupportedHardware
                                     userInfo:@{
                                       NSLocalizedDescriptionKey : @"Unsupported hardware"
                                     }];
        return;
      case CBCentralManagerStateUnauthorized:
        localError = [NSError errorWithDomain:kCBDriverErrorDomain
                                         code:CBDriverErrorUnauthorized
                                     userInfo:@{
                                       NSLocalizedDescriptionKey : @"Unauthorized"
                                     }];
        return;
      default:
        break;
    }
    self.scanUuids = uuids;
    self.baseUuid = baseUuid;
    self.maskUuid = maskUuid;
    self.onDiscoveredHandler = handler;
    self.isScanning = YES;
    if ([self isHardwarePoweredOn]) {
      [self.central scanForPeripheralsWithServices:self.scanUuids
                                           options:@{
                                             CBCentralManagerScanOptionAllowDuplicatesKey : @NO
                                           }];
    }
  });
  if (localError) {
    if (error) {
      *error = localError;
    }
    return NO;
  }
  return YES;
}

- (void)stopScan {
  CBDispatchSync(self.queue, ^{
    if ([self isHardwarePoweredOn]) {
      [self.central stopScan];
      for (CBPeripheralScan *scan in self.scans) {
        [self.central cancelPeripheralConnection:scan.peripheral];
      }
    }
    self.isScanning = NO;
    self.scanUuids = nil;
    self.baseUuid = nil;
    self.maskUuid = nil;
    self.onDiscoveredHandler = nil;
    // We don't just call _initDiscovery because that removes our cache of scanned peripherals
    // which we want to keep.
  });
}

- (void)cancelPendingScans {
  for (CBPeripheralScan *scan in self.scans.allObjects) {
    [self peripheralScanDidComplete:scan];
  }
}

#pragma mark - CBCentralManagerDelegate callbacks

- (void)centralManagerDidUpdateState:(CBCentralManager *)central {
  switch (central.state) {
    case CBCentralManagerStatePoweredOn:
      CBInfoLog(@"CBCentralManagerStateOn");
      if (self.isScanning) {
        [self.central scanForPeripheralsWithServices:self.scanUuids options:nil];
      }
      break;
    case CBCentralManagerStatePoweredOff:
      CBInfoLog(@"CBCentralManagerStateOff");
      [self cancelPendingScans];
      break;
    case CBCentralManagerStateResetting:
      CBInfoLog(@"CBCentralManagerStateResetting");
      [self cancelPendingScans];
      break;
    case CBCentralManagerStateUnauthorized:
      CBInfoLog(@"CBCentralManagerStateUnauthorized");
      [self cancelPendingScans];
      break;
    case CBCentralManagerStateUnknown:
      CBInfoLog(@"CBCentralManagerStateUnknown");
      [self cancelPendingScans];
      break;
    case CBCentralManagerStateUnsupported:
      CBInfoLog(@"CBCentralManagerStateUnsupported");
      [self cancelPendingScans];
      break;
  }
}

- (BOOL)canHandleDiscovery:(CBPeripheral *)peripheral {
  if (![self isHardwarePoweredOn]) {
    CBPeripheralScan *existingScan = [self scanForPeripheral:peripheral];
    if (existingScan) {
      // Unlikely but just in case....
      CBErrorLog(@"Discovered peripheral with existing scan %@ but hardware is now off, dropping",
                 existingScan);
      [self peripheralScanDidComplete:existingScan];
      // Remove from cache -- we're powered off so something weird is going on
      [self.scans removeObject:existingScan];
    }
    return NO;
  }
  if (!self.isScanning) {
    CBErrorLog(@"Discovered peripheral even though we're not scanning -- "
               @"turning off hardware scan");
    // Make sure this isn't a delayed message after hardware getting turned off.
    if ([self isHardwarePoweredOn]) {
      [self.central stopScan];
    }
    return NO;
  }
  return YES;
}

- (NSSet *)targetServiceUuidsInAdData:(NSDictionary<NSString *, id> *_Nonnull)adData {
  // Extract and find service UUIDs that we care about
  NSMutableSet *serviceUuids = [NSMutableSet new];
  NSArray<CBUUID *> *dataServiceUuids = adData[CBAdvertisementDataServiceUUIDsKey];
  if (dataServiceUuids) [serviceUuids addObjectsFromArray:dataServiceUuids];
  NSArray<CBUUID *> *overflowServiceUuids = adData[CBAdvertisementDataOverflowServiceUUIDsKey];
  if (overflowServiceUuids) [serviceUuids addObjectsFromArray:overflowServiceUuids];
  NSSet *matchingUuids = [self filterMatchingUUIDs:serviceUuids];
  return matchingUuids;
}

- (void)centralManager:(CBCentralManager *)central
 didDiscoverPeripheral:(CBPeripheral *)peripheral
     advertisementData:(NSDictionary<NSString *, id> *)adData
                  RSSI:(NSNumber *)RSSI {
  if (![self canHandleDiscovery:peripheral]) {
    return;
  }
  CBDebugLog(@"Discovered peripheral %@ with ad data %@", peripheral, adData);
  NSSet *serviceUuids = [self targetServiceUuidsInAdData:adData];
  if (!serviceUuids.count) {
    CBDebugLog(@"No matching UUIDs -- ignoring discovered peripheral with service uuids %@",
               serviceUuids);
    return;
  }
  CBPeripheralScan *existingScan = [self scanForPeripheral:peripheral];
  if (existingScan) {
    // Are there any new UUIDs being advertised?
    NSMutableSet *newUuids = [serviceUuids mutableCopy];
    [newUuids minusSet:[existingScan seenUUIDs]];
    if (!newUuids.count) {
      // Nothing new -- ignore the duplicate notice.
      CBDebugLog(@"No new service UUIDs -- ignoring");
      return;
    }
    // There are new things we haven't seen before... so we either need to restart this
    // scan or if this scan is in the middle of discovering services then we can safely
    // ignore this (since it's about to see the same thing).
    if (!existingScan.delegate || existingScan.serviceScans.count ||
        // We're still waiting for discover services to come back.. make sure it isn't hung-up
        fabs(existingScan.lastActivity.timeIntervalSinceNow) > kMaxTimeForDiscoverServices) {
      // We have existing service scans... so we can safely restart this scan. Remove the old
      // and we'll create a new one below.
      CBInfoLog(@"Restarting scan of peripheral: %@", existingScan);
      existingScan.peripheral.delegate = nil;
      [self.scans removeObject:existingScan];
    } else {
      // We're waiting for the services to be discovered -- ignore the rotation.
      CBDebugLog(@"Waiting for peripheral to have its services discovered -- ignoring ad");
      return;
    }
  }
  // Start the scan by connecting to it
  CBPeripheralScan *scan =
      [[CBPeripheralScan alloc] initWithPeripheral:peripheral rssi:RSSI delegate:self];
  [self.scans addObject:scan];
  switch (peripheral.state) {
    case CBPeripheralStateConnected:
      [scan start:self.scanUuids];
      break;
    case CBPeripheralStateConnecting:
      break;
    default:  // Disconnected or Disconnecting (depending on SDK)
      CBDebugLog(@"Connecting to peripheral %@", peripheral);
      [central connectPeripheral:peripheral options:nil];
      break;
  }
}

- (void)centralManager:(CBCentralManager *)central didConnectPeripheral:(CBPeripheral *)peripheral {
  CBInfoLog(@"didConnectPeripheral: %@", peripheral);
  CBPeripheralScan *scan = [self scanForPeripheral:peripheral];
  if (!scan) {
    CBInfoLog(@"Peripheral not queued -- ignoring");
    if ([self isHardwarePoweredOn]) {
      [central cancelPeripheralConnection:peripheral];
    }
    return;
  }
  if (!self.isScanning || ![self isHardwarePoweredOn]) {
    CBInfoLog(@"Dropping connected peripheral: isScanning=%d isHardwarePoweredOn=%d",
              self.isScanning, [self isHardwarePoweredOn]);
    [self peripheralScanDidComplete:scan];
    return;
  }
  [scan start:self.scanUuids];
}

- (void)centralManager:(CBCentralManager *)central
    didFailToConnectPeripheral:(CBPeripheral *)peripheral
                         error:(nullable NSError *)error {
  CBErrorLog(@"didFailToConnectPeripheral %@ error %@", peripheral, error);
  if (!self.isScanning) {
    CBInfoLog(@"Not scanning anymore -- disconnecting peripheral");
  }
  CBPeripheralScan *scan = [self scanForPeripheral:peripheral];
  if (!scan) {
    CBInfoLog(@"Peripheral not queued -- ignoring");
    return;
  }
  CBErrorLog(@"Unable to read characteristics for peripheral %@", peripheral);
  [self peripheralScanDidComplete:scan];
}

- (void)centralManager:(CBCentralManager *)central
    didDisconnectPeripheral:(CBPeripheral *)peripheral
                      error:(nullable NSError *)error {
  CBInfoLog(@"didDisconnectPeripheral %@ error %@", peripheral, error);
  CBPeripheralScan *scan = [self scanForPeripheral:peripheral];
  if (!scan) {
    CBDebugLog(@"Peripheral not queued -- ignoring");
    return;
  }
  if (scan.completedScans.count != scan.serviceScans.count) {
    CBInfoLog(@"Lost connection to peripheral mid-query");
    [self peripheralScanDidComplete:scan];
  }
}

#pragma mark - Driver Util

- (CBPeripheralScan *_Nullable)scanForPeripheral:(CBPeripheral *_Nonnull)peripheral {
  for (CBPeripheralScan *scan in self.scans) {
    if ([scan.peripheral isEqual:peripheral]) {
      return scan;
    }
  }
  return nil;
}

- (NSSet *)filterMatchingUUIDs:(NSSet *)allUuids {
  NSMutableSet *matchingUuids = [NSMutableSet new];
  for (CBUUID *uuid in allUuids) {
    if ([self uuidMatchesScanFilter:uuid]) {
      [matchingUuids addObject:uuid];
    }
  }
  return matchingUuids;
}

- (BOOL)uuidMatchesScanFilter:(CBUUID *)serviceUuid {
  if ([self.scanUuids containsObject:serviceUuid]) {
    CBDebugLog(@"Found matching service %@", serviceUuid.UUIDString);
    return YES;
  }
  if (self.baseUuid && self.maskUuid) {
    if (serviceUuid.data.length != self.maskUuid.data.length) {
      CBDebugLog(@"Not applying mask to different length UUID %@", serviceUuid);
      return NO;
    }
    // Find UUIDs that match base with the mask applied
    NSMutableData *maskedUuidData = [NSMutableData dataWithData:serviceUuid.data];
    const char *maskBytes = self.maskUuid.data.bytes;
    char *maskedUuidBytes = maskedUuidData.mutableBytes;
    for (int i = 0; i < self.maskUuid.data.length; i++) {
      maskedUuidBytes[i] = maskedUuidBytes[i] & maskBytes[i];
    }
    CBUUID *maskedUuid = [CBUUID UUIDWithData:maskedUuidData];
    if ([maskedUuid isEqual:self.baseUuid]) {
      CBDebugLog(@"Found service %@ via mask", serviceUuid.UUIDString);
      return YES;
    }
  }
  return NO;
}

- (NSString *)debugDescription {
  if (!self.queue) return @"[CBScanningDriver missing queue -- broken state]";
  __block NSString *out = nil;
  CBDispatchSync(self.queue, ^{
    out = [NSString stringWithFormat:@"[CBScanningDriver isHardwareOn=%d isScanning=%d "
                                     @"scanForUuids=%@, scans=%@]",
                                     self.isHardwarePoweredOn, self.isScanning, self.scanUuids,
                                     self.scans];
  });
  return out;
}

#pragma mark - Scan Delegates

- (void)peripheralScanDidComplete:(CBPeripheralScan *)peripheralScan {
  CBInfoLog(@"Done scanning peripheral %@", peripheralScan.peripheral);
  peripheralScan.peripheral.delegate = nil;
  if ([self isHardwarePoweredOn]) {
    [self.central cancelPeripheralConnection:peripheralScan.peripheral];
  }
}

- (void)serviceScanDidComplete:(CBPeripheralServiceScan *)serviceScan {
  CBInfoLog(@"serviceScanDidComplete: %@", serviceScan);
  if (self.onDiscoveredHandler) {
    int rssi = serviceScan.rssi.intValue;
    // From:
    // https://developer.apple.com/library/mac/documentation/IOBluetooth/Reference/IOBluetoothDevice_reference/#//apple_ref/occ/instm/IOBluetoothDevice/RSSI
    // "If the value cannot be read (e.g. the device is disconnected) or is not available on a
    // module, a value of +127 will be returned."
    if (rssi == 127) {
      CBDebugLog(@"RSSI of +127 found, reporting at 0 to Go");
      rssi = 0;
    }
    CBDebugLog(@"Notifying of service %@ with characteristics %@ and rssi of %d",
               serviceScan.service.UUID, serviceScan.characteristics, rssi);
    self.onDiscoveredHandler(serviceScan.service.UUID, serviceScan.characteristics, rssi);
  }
}

- (BOOL)isHardwarePoweredOn {
  return self.central.state == CBCentralManagerStatePoweredOn;
}

@end

#pragma mark - Peripheral Scoped Scanning

@implementation CBPeripheralScan

- (id)initWithPeripheral:(CBPeripheral *_Nonnull)peripheral
                    rssi:(NSNumber *)rssi
                delegate:(id<CBPeripheralScanDelegate, CBPeripheralServiceScanDelegate>)delegate {
  if (self = [super init]) {
    [self updateLastActivity];
    self.peripheral = peripheral;
    // This is potentially a cycle in 9.0 (weak -> assign in API change), however we are tracking
    // this scan and when it's done we will undo the cycle.
    self.peripheral.delegate = self;
    self.delegate = delegate;
    self.rssi = rssi;
    self.serviceScans = [NSMutableSet new];
  }
  return self;
}

- (void)start:(NSArray<CBUUID *> *_Nullable)scanUuids {
  if (self.peripheral.state != CBPeripheralStateConnected) {
    CBErrorLog(@"Peripheral not connected -- can't discover services on %@", self.peripheral);
    [self.delegate peripheralScanDidComplete:self];
    return;
  }
  [self updateLastActivity];
  CBDebugLog(@"Discovering services on %@", self.peripheral);
  if (scanUuids.count) {
    [self.peripheral discoverServices:scanUuids];
  } else {
    [self.peripheral discoverServices:nil];
  }
}

#pragma mark - CBPeripheralDelegate callbacks

- (void)peripheral:(CBPeripheral *)peripheral didDiscoverServices:(nullable NSError *)error {
  [self updateLastActivity];
  if (error) {
    CBErrorLog(@"didDiscoverServices on %@ error %@", peripheral, error);
    [self.delegate peripheralScanDidComplete:self];
    return;
  }
  if (!peripheral.services.count) {
    CBDebugLog(@"Peripheral %@ has no services", peripheral);
    [self.delegate peripheralScanDidComplete:self];
    return;
  }
  // Filter for Vanadium services
  NSMutableSet<CBService *> *discoveredServices = [NSMutableSet new];
  NSMutableSet<CBUUID *> *discoveredUuids = [NSMutableSet new];
  for (CBService *service in peripheral.services) {
    if ([self.delegate uuidMatchesScanFilter:service.UUID]) {
      [discoveredServices addObject:service];
      [discoveredUuids addObject:service.UUID];
    }
  }
  NSMutableSet *knownUuids = [self seenUUIDs];
  // Find the ones we previously knew about, but aren't part of the current set.
  [knownUuids minusSet:discoveredUuids];
  for (CBUUID *expiredService in knownUuids) {
    CBPeripheralServiceScan *existingScan = [self scanForServiceUUID:expiredService];
    if (existingScan) {
      CBInfoLog(@"Removing service that isn't part of the current discovered set: %@",
                existingScan);
      [self.serviceScans removeObject:existingScan];
      [self.completedScans removeObject:existingScan];
    }
  }
  // Scan anything new
  for (CBService *service in discoveredServices) {
    CBPeripheralServiceScan *existingScan = [self scanForServiceUUID:service.UUID];
    if (existingScan) {
      CBDebugLog(@"Ignoring known service UUID %@", service.UUID);
      return;
    }
    CBDebugLog(@"Starting scan for new service UUID %@", service.UUID);
    CBPeripheralServiceScan *serviceScan =
        [[CBPeripheralServiceScan alloc] initWithService:service rssi:self.rssi delegate:self];
    [self.serviceScans addObject:serviceScan];
    [serviceScan start];
  }
}

- (void)peripheral:(CBPeripheral *)peripheral
 didModifyServices:(NSArray<CBService *> *)invalidatedServices {
  [self updateLastActivity];
  CBInfoLog(@"peripheral %@ didModifyServices by invalidating %@", peripheral, invalidatedServices);
  NSMutableArray *invalidatedScans = [NSMutableArray new];
  for (CBPeripheralServiceScan *scan in self.serviceScans) {
    for (CBService *invalidatedService in invalidatedServices) {
      if ([scan.service.UUID isEqual:invalidatedService.UUID]) {
        // Because the upper chain might modify self.serviceScans we must first extract this scan
        // before we can safety end the scan.
        [invalidatedScans addObject:scan];
        break;  // out of inner loop only
      }
    }
  }
  for (CBPeripheralServiceScan *scan in invalidatedScans) {
    CBDebugLog(@"Invalidated scan %@", scan);
    [self serviceScanDidComplete:scan];
  }
}

- (void)peripheral:(CBPeripheral *)peripheral
    didDiscoverCharacteristicsForService:(CBService *)service
                                   error:(nullable NSError *)error {
  [self updateLastActivity];
  // Forward to the right scan
  for (CBPeripheralServiceScan *scan in self.serviceScans) {
    if ([scan.service.UUID isEqual:service.UUID]) {
      [scan peripheral:peripheral didDiscoverCharacteristicsForService:service error:error];
      break;
    }
  }
}

- (void)peripheral:(CBPeripheral *)peripheral
    didUpdateValueForCharacteristic:(CBCharacteristic *)characteristic
                              error:(nullable NSError *)error {
  [self updateLastActivity];
  // Forward to the right scan
  for (CBPeripheralServiceScan *scan in self.serviceScans) {
    if ([scan.service.UUID isEqual:characteristic.service.UUID]) {
      [scan peripheral:peripheral didUpdateValueForCharacteristic:characteristic error:error];
      break;
    }
  }
}

#pragma mark - Scan delegate

- (void)serviceScanDidComplete:(CBPeripheralServiceScan *)serviceScan {
  [self.completedScans addObject:serviceScan];
  [self.delegate serviceScanDidComplete:serviceScan];
  if (self.completedScans.count == self.serviceScans.count) {
    // We're done with this periperal scan
    [self.delegate peripheralScanDidComplete:self];
  }
}

- (void)updateLastActivity {
  self.lastActivity = [NSDate date];
}

#pragma mark - CBPeripheralScan Util

- (NSMutableSet<CBUUID *> *_Nonnull)seenUUIDs {
  NSMutableSet<CBUUID *> *set = [NSMutableSet new];
  for (CBPeripheralServiceScan *scan in self.serviceScans) {
    [set addObject:scan.service.UUID];
  }
  for (CBPeripheralServiceScan *scan in self.completedScans) {
    [set addObject:scan.service.UUID];
  }
  return set;
}

- (CBPeripheralServiceScan *_Nullable)scanForServiceUUID:(CBUUID *)serviceUUID {
  for (CBPeripheralServiceScan *scan in self.serviceScans) {
    if ([scan.service.UUID isEqual:serviceUUID]) {
      return scan;
    }
  }
  return nil;
}

- (NSString *)description {
  NSString *format = @"[CBPeripheralScan peripheral=%@ rssi=%@ serviceScans=%@ completedScans=%@]";
  return [NSString
      stringWithFormat:format, self.peripheral, self.rssi, self.serviceScans, self.completedScans];
}

@end

#pragma mark - Peripheral's Service Scoped Scanning

@implementation CBPeripheralServiceScan

- (id)initWithService:(CBService *_Nonnull)service
                 rssi:(NSNumber *)rssi
             delegate:(id<CBPeripheralServiceScanDelegate>)delegate {
  if (self = [super init]) {
    self.service = service;
    self.delegate = delegate;
    self.rssi = rssi;
    self.queryingCharacteristics = [NSMutableSet new];
    self.characteristics = [NSMutableDictionary new];
  }
  return self;
}

- (void)start {
  NSArray<CBUUID *> *uuids = [CBPeripheralServiceScan possibleCharacteristicUUIDs];
  if (self.service.peripheral.state != CBPeripheralStateConnected) {
    CBErrorLog(@"Can't discover characteristics for service %@ -- we're not connected",
               self.service.UUID);
    [self.delegate serviceScanDidComplete:self];
    return;
  }
  [self.delegate updateLastActivity];
  [self.service.peripheral discoverCharacteristics:uuids forService:self.service];
}

#pragma mark - CBPeripheralDelegate callbacks relevant to this service

- (void)peripheral:(CBPeripheral *)peripheral
    didDiscoverCharacteristicsForService:(CBService *)service
                                   error:(nullable NSError *)error {
  CBDebugLog(@"didDiscoverCharacteristicsForService %@ error %@", service.UUID, error);
  [self.delegate updateLastActivity];
  if (error) {
    CBErrorLog(@"Unable to discover characteristics for service: %@", service.UUID);
    [self.delegate serviceScanDidComplete:self];
    return;
  }
  if (!service.characteristics.count) {
    CBInfoLog(@"No characteristics for service %@", service.UUID);
    [self.delegate serviceScanDidComplete:self];
  } else {
    CBInfoLog(@"Getting characteristics for service %@", service.UUID);
    for (CBCharacteristic *characteristic in service.characteristics) {
      if ((characteristic.properties & CBCharacteristicPropertyRead) == 0) {
        CBDebugLog(@"Skipping un-readable characteristic %@", characteristic);
      } else {
        [self.queryingCharacteristics addObject:characteristic];
        CBDebugLog(@"Reading value for characteristic %@", characteristic);
        [peripheral readValueForCharacteristic:characteristic];
      }
    }
  }
}

- (void)peripheral:(CBPeripheral *)peripheral
    didUpdateValueForCharacteristic:(CBCharacteristic *)characteristic
                              error:(nullable NSError *)error {
  CBDebugLog(@"didUpdateValueForCharacteristic %@ error %@", characteristic, error);
  [self.delegate updateLastActivity];
  if (error) {
    CBErrorLog(@"Unable to get value for characteristic %@ due to error %@", characteristic, error);
  } else {
    NSData *data = characteristic.value;
    if (data.length) {
      self.characteristics[characteristic.UUID] = data;
    }
  }
  // Are we done with this service scan?
  [self.queryingCharacteristics removeObject:characteristic];
  if (!self.queryingCharacteristics.count) {
    [self.delegate serviceScanDidComplete:self];
  } else {
    CBDebugLog(@"Waiting on %d characteristics", self.queryingCharacteristics.count);
  }
}

#pragma mark - CBPeripheralServiceScan Util

- (NSString *)description {
  return [NSString stringWithFormat:@"[CBPeripheralServiceScan service=%@ rssi=%@ "
                                    @"queryingCharacteristics=%@ characteristics=%@]",
                                    self.service.UUID, self.rssi, self.queryingCharacteristics,
                                    self.characteristics];
}

+ (NSArray<CBUUID *> *)possibleCharacteristicUUIDs {
  static NSArray *_cached = nil;
  if (!_cached) {
    // Compute all possible UUIDs
    NSMutableArray *uuids = [NSMutableArray new];
    for (int i = 0; i < kMaxNumPackedServices; i++) {
      for (int j = 0; j < kMaxNumPackedCharacteristicsPerService; j++) {
        NSString *uuid = [NSString stringWithFormat:kPackedCharacteristicUuidFmt, i, j];
        [uuids addObject:[CBUUID UUIDWithString:uuid]];
      }
    }
    _cached = uuids;
  }
  return _cached;
}

@end
