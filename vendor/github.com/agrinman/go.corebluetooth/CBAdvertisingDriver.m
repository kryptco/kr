// +build darwin,cgo
// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import "CBAdvertisingDriver.h"
#import "CBDriver.h"
#import "CBLog.h"
#import "CBUtil.h"

static const void *kRotateAdDelay = &kRotateAdDelay;
// This is exported from go
extern void v23_corebluetooth_go_data_received(const char *_Nonnull data, int dataLength);

static const char offByte = 0;
static const char pingByte = 1;

@interface CBAdvertisingDriver ()
@property(nonatomic, assign) AdvertisingState advertisingState;
@property(nonatomic, assign) BOOL rotateAd;
@property(nonatomic, strong) NSDate *requestedAdvertisingStart;
/** advertisedServices is the same as the keys of services, but the ordering maintained here is
 how we do this rotation business. */
@property(nonatomic, strong) NSMutableArray *_Nonnull advertisedServices;
/** All added services by uuid */
@property(nonatomic, strong) NSMutableDictionary *_Nonnull services;
@property(nonatomic, strong)
    NSMutableDictionary *_Nonnull addServiceHandlers;
@property(nonatomic, strong) NSMutableDictionary
    *_Nonnull serviceCharacteristics;
@property(nonatomic, strong) NSMutableArray*_Nonnull writeQueue;
@property(nonatomic, strong) NSData*_Nullable lastMessage;
@property(nonatomic, assign) int centralMTU;
@end

@implementation CBAdvertisingDriver


- (id _Nullable)initWithQueue:(dispatch_queue_t _Nonnull)queue {
  if (self = [super init]) {
    self.queue = queue;
    self.peripheral =
        [[CBPeripheralManager alloc] initWithDelegate:self
                                                queue:self.queue
                                              options:@{
                                                CBPeripheralManagerOptionShowPowerAlertKey : @YES
                                              }];
    self.services = [NSMutableDictionary new];
    self.serviceCharacteristics = [NSMutableDictionary new];
    self.addServiceHandlers = [NSMutableDictionary new];
    self.advertisedServices = [NSMutableArray new];
    self.advertisingState = AdvertisingStateNotAdvertising;
	self.lastMessage = nil;
    self.rotateAdDelay = 1;
	self.centralMTU = 20;
	self.writeQueue = [NSMutableArray new];
  }
  return self;
}

- (void)dealloc {
  CBDispatchSync(self.queue, ^{
    self.peripheral.delegate = nil;
    if (self.isHardwarePoweredOn) {
      [self.peripheral stopAdvertising];
      [self.peripheral removeAllServices];
    }
  });
}

- (void)addService:(CBUUID *_Nonnull)uuid
   characteristics:(NSDictionary *_Nonnull)characteristics
          callback:(CBAddServiceHandler _Nonnull)handler {
  // Allow shutdown in between to occur.
    CBDebugLog(@"addService");
  __weak typeof(self) this = self;
  dispatch_async(self.queue, ^{
    if (!this) {
    CBDebugLog(@"addService !this");
      return;
    }
    if (this.services[uuid]) {
    CBInfoLog(@"addService already added");
      handler(uuid, [NSError errorWithDomain:kCBDriverErrorDomain
                                        code:CBDriverErrorServiceAlreadyAdded
                                    userInfo:@{
                                      NSLocalizedDescriptionKey : @"Service already added"
                                    }]);
      return;
    }
    //CBMutableService *service =
        //[CBMutableService cb_mutableService:uuid withReadOnlyCharacteristics:characteristics];
    CBMutableService *service =
	[[CBMutableService alloc] initWithType:uuid primary:YES];
	CBUUID *charUUID = [CBUUID UUIDWithString:@"20F53E48-C08D-423A-B2C2-1C797889AF24"];
	CBMutableCharacteristic *characteristic = [[CBMutableCharacteristic alloc] initWithType:charUUID properties:CBCharacteristicPropertyWrite|CBCharacteristicPropertyRead|CBCharacteristicPropertyNotify value:nil permissions:CBAttributePermissionsWriteable|CBAttributePermissionsReadable];
	service.characteristics = @[characteristic];
    // Save the service & characteristics for later reads
    this.services[uuid] = service;
    this.serviceCharacteristics[uuid] = characteristic;
    // Insert this as the next service advertised
    [this.advertisedServices insertObject:uuid atIndex:0];
    CBDebugLog(@"Advertised services is now %@", this.advertisedServices);
    // We can't do anything if we're not powered on -- but it's queued to be added whenever it is
    if (this.isHardwarePoweredOn) {
      this.addServiceHandlers[uuid] = handler;
      [this.peripheral addService:service];
      CBInfoLog(@"Added service %@", service);
      [this stopAdvertising];
      [this startAdvertising];
    } else {
      // Tell handler it was a success since we can't know any true error until the hardware is
      // ready.
      CBInfoLog(@"Queued service %@", service);
      handler(service.UUID, nil);
    }
  });
}

- (void)writeData:(NSData*_Nonnull)data 
		 callback:(CBBoolHandler _Nullable)handler {
  __weak typeof(self) this = self;
  dispatch_async(self.queue, ^{
    if (!this) {
    CBDebugLog(@"writeData !this");
      return;
    }
	this.lastMessage = [NSData dataWithData:data];
	NSMutableArray* splitMessage = [this splitMessage:data];
    CBDebugLog(@"writeData %d bytes split into %d messages", data.length, splitMessage.count);
	BOOL result = YES;
	for (NSData* split in splitMessage) {
		result &= [this writeDataRaw:split];
	}
	if (handler != nil ) {
		handler(result, nil);
	}
    });
}

-(NSMutableArray*)splitMessage:(NSData*_Nonnull)message {
	NSMutableArray* split = [NSMutableArray new];
	int offset = 0;
	int msgBlockSize = self.centralMTU - 1;
	if (message.length / msgBlockSize > 255) {
		CBErrorLog(@"message of length %d too long", message.length);
		return split;
	}
	for (char n = message.length / msgBlockSize; n >= 0; n--) {
		int endIndex = MIN(message.length, offset + msgBlockSize);
		NSMutableData* block = [NSMutableData new];
		[block appendBytes:&n length: 1];
		[block appendData:[message subdataWithRange:NSMakeRange(offset, endIndex - offset)]];
		offset += msgBlockSize;
		[split addObject:block];
	}
	return split;
}

//	Message chunk already split by centralMTU
- (BOOL)writeDataRaw:(NSData*_Nonnull)data {
	BOOL result = NO;
	[self.writeQueue addObject:data];
	return [self tryWrite];
}

- (BOOL)tryWrite {
	BOOL result = NO;
	NSData* dataToWrite = self.writeQueue[0];
	for (CBMutableService *service in self.services.allValues) {
		result |= [self.peripheral updateValue:dataToWrite forCharacteristic:self.serviceCharacteristics[service.UUID] onSubscribedCentrals:nil];
	}
	if (result) {
		[self.writeQueue removeObjectAtIndex:0];
	}
	return result;
}

- (void)peripheralManagerIsReadyToUpdateSubscribers:(CBPeripheralManager *_Nonnull)peripheral {
    CBDebugLog(@"peripheralManagerIsReadyToUpdateSubscribers");
	while ([self.writeQueue count] > 0 && [self tryWrite]) {
		CBDebugLog(@"writing from queue");
	}
}

- (NSUInteger)serviceCount {
  return self.services.count;
}

- (void)removeService:(CBUUID *_Nonnull)uuid {
  CBInfoLog(@"removeService: %@", uuid.UUIDString);
  CBDispatchSync(self.queue, ^{
	NSMutableData* offMsg = [NSMutableData new];
	[offMsg appendBytes:&offByte length: 1];
	[self writeDataRaw:offMsg];
    CBMutableService *service = self.services[uuid];
    if (!service) {
      return;
    }
    if (self.isHardwarePoweredOn) {
      [self.peripheral removeService:service];
    }
	[self.services removeObjectForKey:uuid];
	[self.serviceCharacteristics removeObjectForKey:uuid];
    [self.advertisedServices removeObject:uuid];
	[self.addServiceHandlers removeObjectForKey:uuid];
    // Undo the service from our current ads
    [self stopAdvertising];
    // startAdvertising will check the current count before enabling
    [self startAdvertising];
  });
}

- (void)addAllServices {
  [self threadSafetyCheck];
  if (!self.isHardwarePoweredOn) {
    CBErrorLog(@"Can't add all services -- BLE hardware state isn't powered on");
    return;
  }
  [self.peripheral removeAllServices];
  for (CBMutableService *service in self.services.allValues) {
    [self.peripheral addService:service];
  }
}

- (void)startAdvertising {
  [self threadSafetyCheck];
  //if (self.advertisingState != AdvertisingStateNotAdvertising) {
    //CBInfoLog(@"Not starting advertising when in state %d", self.advertisingState);
    //return;
  //}
  if (!self.advertisedServices.count) {
    CBDebugLog(@"Nothing to advertise");
    return;
  }
  if (!self.isHardwarePoweredOn) {
    CBInfoLog(@"Not powered on -- can't advertise yet");
    return;
  }
  NSDictionary *ad = @{CBAdvertisementDataServiceUUIDsKey : self.advertisedServices, CBAdvertisementDataLocalNameKey: @"krsshagent"};
  self.advertisingState = AdvertisingStateStarting;
  CBInfoLog(@"startAdvertising %@", self.advertisedServices);
  [self invalidateRotateAdTimer];
  // When we get the callback advertising started then we reschedule rotation.
  self.requestedAdvertisingStart = [NSDate date];
  [self.peripheral startAdvertising:ad];
}

- (void)stopAdvertising {
  [self threadSafetyCheck];
  CBInfoLog(@"stopAdvertising");
  self.advertisingState = AdvertisingStateNotAdvertising;
  if (self.isHardwarePoweredOn) {
    [self.peripheral stopAdvertising];
  }
  [self invalidateRotateAdTimer];
}

- (void)scheduleRotateAdTimer:(NSTimeInterval)delay {
  self.rotateAd = YES;
  __weak typeof(self) this = self;
  dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(delay * NSEC_PER_SEC)), self.queue, ^{
    [this rotateAdTimerDidFire];
  });
}

- (void)invalidateRotateAdTimer {
  self.rotateAd = NO;
}

- (void)rotateAdTimerDidFire {
  [self threadSafetyCheck];
  if (!self.rotateAd) return;
  switch (self.advertisingState) {
    case AdvertisingStateNotAdvertising:
      CBDebugLog(@"Not advertising - expiring timer");
      [self invalidateRotateAdTimer];
      return;
    case AdvertisingStateStarting:
      CBDebugLog(@"Advertising still starting, ignoring timer");
      return;
    case AdvertisingStateAdvertising:
      if (self.advertisedServices.count < 2) {
        CBDebugLog(@"Not enough services to rotate bluetooth ad - expiring timer");
        [self invalidateRotateAdTimer];
        return;
      }
      break;
  }

  // Rotate services
  CBUUID *head = [self.advertisedServices firstObject];
  [self.advertisedServices removeObjectAtIndex:0];
  [self.advertisedServices addObject:head];
  [self stopAdvertising];
  [self startAdvertising];  // This will reschedule the next timer if needed
}

- (void)peripheralManagerDidUpdateState:(CBPeripheralManager *)peripheral {
  switch (peripheral.state) {
    case CBPeripheralManagerStatePoweredOn:
      CBInfoLog(@"CBPeripheralManagerStatePoweredOn");
      if (self.services.count > 0 && self.advertisingState == AdvertisingStateNotAdvertising) {
		  [self addAllServices];
		  [self startAdvertising];
      }
      break;
    case CBPeripheralManagerStatePoweredOff:
      CBInfoLog(@"CBPeripheralManagerStatePoweredOff");
	  //	stopping advertising here seems to leave around zombie bluetooth advertisements
      //[self stopAdvertising];
      break;
    case CBPeripheralManagerStateResetting:
      CBInfoLog(@"CBPeripheralManagerStateResetting");
      [self stopAdvertising];
      break;
    case CBPeripheralManagerStateUnauthorized:
      CBInfoLog(@"CBPeripheralManagerStateUnauthorized");
      [self stopAdvertising];
      break;
    case CBPeripheralManagerStateUnknown:
      CBInfoLog(@"CBPeripheralManagerStateUnknown");
      [self stopAdvertising];
      break;
    case CBPeripheralManagerStateUnsupported:
      CBInfoLog(@"CBPeripheralManagerStateUnsupported");
      [self stopAdvertising];
      break;
  }
}

- (void)peripheralManagerDidStartAdvertising:(CBPeripheralManager *)peripheral
                                       error:(nullable NSError *)error {
  [self threadSafetyCheck];
  NSTimeInterval responseTime = fabs([self.requestedAdvertisingStart timeIntervalSinceNow]);
  if (error) {
    CBErrorLog(@"Error advertising: %@", error);
    [self stopAdvertising];
    return;
  }
  CBDebugLog(@"Now advertising - %2.f msec to start", responseTime * 1000);
  self.advertisingState = AdvertisingStateAdvertising;
  if (self.advertisedServices.count > 1) {
    // Guarantee at least 100msec for battery
    NSTimeInterval howSoon = MAX(0.1, self.rotateAdDelay - responseTime);
    [self scheduleRotateAdTimer:howSoon];
  }
}

- (void)peripheralManager:(CBPeripheralManager *)peripheral
            didAddService:(CBService *)service
                    error:(nullable NSError *)error {
  [self threadSafetyCheck];
  CBAddServiceHandler handler = self.addServiceHandlers[service.UUID];
  [self.addServiceHandlers removeObjectForKey:service.UUID];
  if (!handler) {
    CBDebugLog(@"Missing handler for adding service %@ with error %@", service, error);
    return;
  }
  handler(service.UUID, error);
}

- (void)peripheralManager:(CBPeripheralManager *)peripheral
                         central:(CBCentral *)central
    didSubscribeToCharacteristic:(CBCharacteristic *)characteristic {
  __weak typeof(self) this = self;
  dispatch_async(self.queue, ^{
    if (!this) {
	  CBErrorLog(@"didSubscribe !this");
      return;
    }
	//	Apple example code always uses notify MTU of 20
	//this.centralMTU = central.maximumUpdateValueLength;
	CBInfoLog(@"central %@ didSubscribeToCharacteristic %@", central, characteristic);
	if (this.lastMessage != nil) {
	    [this writeData:this.lastMessage callback:nil];
	}
  });
}

- (void)peripheralManager:(CBPeripheralManager *)peripheral
                             central:(CBCentral *)central
    didUnsubscribeFromCharacteristic:(CBCharacteristic *)characteristic {
  CBInfoLog(@"central %@ didUnsubscribeFromCharacteristic %@", central, characteristic);
}

- (void)peripheralManager:(CBPeripheralManager *)peripheral
    didReceiveReadRequest:(CBATTRequest *)request {
  CBErrorLog(@"didReceiveReadRequest %@", request);
  CBCharacteristic *characteristic = self.serviceCharacteristics[request.characteristic.service.UUID];
  if (!characteristic) {
    [peripheral respondToRequest:request withResult:CBATTErrorAttributeNotFound];
    return;
  }
  NSData *data = characteristic.value;
  if (!data) {
    [peripheral respondToRequest:request withResult:CBATTErrorAttributeNotFound];
    return;
  }
  if (request.offset >= data.length) {
    [peripheral respondToRequest:request withResult:CBATTErrorInvalidOffset];
    return;
  }
  request.value = [data subdataWithRange:NSMakeRange(request.offset, data.length - request.offset)];
  [peripheral respondToRequest:request withResult:CBATTErrorSuccess];
}

- (void)peripheralManager:(CBPeripheralManager *)peripheral
  didReceiveWriteRequests:(NSArray *)requests {
  CBDebugLog(@"didReceiveWriteRequests %@", requests);
  for (CBATTRequest *request in requests) {
	  if (request.value != nil ) {
		  if (request.value.length == 1) {
			  [self processControlMessage:*(const char*)request.value.bytes];
		  } else {
			  v23_corebluetooth_go_data_received(request.value.bytes, request.value.length);
		  }
	  }
    [peripheral respondToRequest:request withResult:CBATTErrorSuccess];
  }
}

- (void)processControlMessage:(const char)byte {
	NSMutableData* pingMsg = [NSMutableData new];
	switch (byte) {
		case pingByte:
			[pingMsg appendBytes:&pingByte length: 1];
			[self writeDataRaw:pingMsg];
			break;
	}
}

- (void)threadSafetyCheck {
  assert(!strcmp(dispatch_queue_get_label(DISPATCH_CURRENT_QUEUE_LABEL),
                 dispatch_queue_get_label(self.queue)));
}

- (BOOL)isHardwarePoweredOn {
  return self.peripheral.state == CBPeripheralManagerStatePoweredOn;
}

- (NSString *)debugDescription {
  if (!self.queue) return @"[CBScanningDriver missing queue -- broken state]";
  __block NSString *out = nil;
  CBDispatchSync(self.queue, ^{
    NSString *state;
    switch (self.advertisingState) {
      case AdvertisingStateAdvertising:
        state = @"advertising";
        break;
      case AdvertisingStateNotAdvertising:
        state = @"not advertising";
        break;
      case AdvertisingStateStarting:
        state = @"starting";
        break;
    }
    out = [NSString stringWithFormat:@"[CBAdvertisingDriver isHardwareOn=%d advertisingState=%@ "
                                     @"rotateAd=%d serviceUuids=%@]",
                                     self.isHardwarePoweredOn, state, self.rotateAd,
                                     self.services.allKeys];
  });
  return out;
}

@end
