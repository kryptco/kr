// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <CoreBluetooth/CoreBluetooth.h>

typedef enum {
  AdvertisingStateNotAdvertising,
  AdvertisingStateStarting,
  AdvertisingStateAdvertising
} AdvertisingState;

/** CBAddServiceHandler is called when a service has/hasn't been added (error set when not added) */
typedef void (^CBAddServiceHandler)(CBUUID *_Nonnull uuid, NSError *_Nullable error);
typedef void (^CBBoolHandler)(BOOL result, NSError *_Nullable error);

/*!
 * This driver implements BLE advertising for Vanadium. CoreBluetooth is very delegate-driven
 * in its design which adds complexity when bridging to synchronous go. For example, we can't
 * synchronously know if starting advertising will return an error. To implement this driver,
 * we allow Obj-C to remain fairly high-level using blocks and rely on the exported C functions
 * (for CGO to call) to translate between objective-c/go.
 *
 * Advertising for Vanadium discovery on CoreBluetooth presents some cross-platform challenges.
 * Unlike Android, iOS is only able to advertise 28 bytes in the foreground [1], which means max one
 * 128-bit service UUID (and some 16-bits, but those aren't useful here). That's because
 * 128-bits = 16 bytes, and 2 services would be 32 bytes and thus over our limit. A Vanadium
 * runtime may have multiple services, each 128-bits, so we have what seems like a problem here.
 * However, iOS provides a prioprietary way of broadcasting more service UUIDs to only other
 * iOS devices using its "overflow services uuids" while in the foreground. OS X does not have
 * access to the overflow area so it only can see the first service of many, like Android.
 *
 * To allow other platforms to see all the service UUIDs we rotate with some adjustable frequency
 * (defaults to one second -- more experimentation needed to determine the "right" number).
 * For example, ["uuid1", "uuid2", "uuid3"] is advertised for one second, then the advertising
 * changes to ["uuid2", "uuid3", "uuid1"], and so on. Android and OS X will be able to eventually
 * observe all the service UUIDs while iOS will always be able to see everything.
 *
 * Note all advertise service UUIDs already will contain a Vanadium stamp (via a base uuid + mask),
 * so all platforms can discover Vanadium devices based on just 1 service UUID and then
 * opportunistically connect in order to retrieve all service UUIDs and their characteristics,
 * regardless of platform.
 *
 * [1] https://developer.apple.com/library/ios/documentation/NetworkingInternetWeb/Conceptual/CoreBluetooth_concepts/BestPracticesForSettingUpYourIOSDeviceAsAPeripheral/BestPracticesForSettingUpYourIOSDeviceAsAPeripheral.html
 *
 */
@interface CBAdvertisingDriver : NSObject<CBPeripheralManagerDelegate>

/** If more than one service is being advertised, CBDriver employs a strategy that rotates the
 ads so that other platforms like Android can discover all service uuids over time. The order of the
 services rotates every ```rotateAdDelay``` seconds.

 Defaults to 1 second.
 */
@property(nonatomic, assign) NSTimeInterval rotateAdDelay;

@property(nonatomic, strong) dispatch_queue_t _Nonnull queue;
@property(nonatomic, strong) CBPeripheralManager *_Nonnull peripheral;

- (id _Nullable)initWithQueue:(dispatch_queue_t _Nonnull)queue;

/** AddService adds a new service to the GATT server with the given service uuid
 and characteristics and starts advertising the service uuid.

 The characteristics will not be changed while it is being advertised.

 There can be multiple services at any given time and it is the driver's
 responsibility to handle multiple advertisements in a compatible way. The CoreBluetooth driver
 does this as follows:

 1. If on iOS, other iOS devices can see all service uuids using Apple's proprietary overflow area

 2. Otherwise (if on a mac, other device is Android, etc) the other device will only be able to
    see 1 service uuid at a time. We rotate through our service uuids every self.rotateAdDelay
    seconds (defaulting to 1) such that all will be eventually visible as the 1 uuid they can see.

 Unlike other discovery plugins, the ble plugin does not allow multiple instances
 of the same service - i.e., the same interface name for now. If we ever need to
 allow it, we also have to pass the instance id for a driver to handle it.

 @param uuid The service UUID
 @param characteristics A dictionary of GATT characteristics that carry the advertisement payload
 @param callback Called on the driver's queue indicating if the service was added successfully.
 */
- (void)addService:(CBUUID *_Nonnull)uuid
   characteristics:(NSDictionary *_Nonnull)characteristics
          callback:(CBAddServiceHandler _Nonnull)handler;

- (void)writeData:(NSData*_Nonnull)data
		 callback:(CBBoolHandler _Nullable)handler;

- (void)peripheralManagerIsReadyToUpdateSubscribers:(CBPeripheralManager *_Nonnull)peripheral;

/** Current number of added services */
- (NSUInteger)serviceCount;

/** RemoveService removes the service from the GATT server and stops advertising the service uuid.
 */
- (void)removeService:(CBUUID *_Nonnull)uuid;

@end
