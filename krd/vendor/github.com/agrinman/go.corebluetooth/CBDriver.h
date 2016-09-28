// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <CoreBluetooth/CoreBluetooth.h>
#import <Foundation/Foundation.h>
#import "CBAdvertisingDriver.h"
#import "CBScanningDriver.h"

extern NSString *_Nonnull kCBDriverErrorDomain;

typedef enum {
  CBDriverErrorUnsupportedHardware = -100,
  CBDriverErrorUnauthorized = -101,
  CBDriverErrorServiceAlreadyAdded = -102,
} CBDriverError;

@interface CBDriver : NSObject

@property(nonatomic, strong) dispatch_queue_t _Nonnull queue;
@property(nonatomic) CBAdvertisingDriver *_Nullable advertisingDriver;
@property(nonatomic) CBScanningDriver *_Nullable scanningDriver;

/** Shared instance for the CBDriver -- only one can ever be alive to correctly use CoreBluetooth.
 */
+ (CBDriver *_Nonnull)instance;

/** Call to remove all services, stop all scans, and remove central/peripheral managers. */
+ (void)shutdown;

@end

#pragma mark - CGO exports

typedef struct {
  char *_Nonnull uuid;
  const void *_Nonnull data;
  int dataLength;
} CBDriverCharacteristicMapEntry;

// Debug String via CGO
char *_Nonnull v23_cbdriver_debug_string();

// Advertising via CGO
BOOL v23_cbdriver_addService(const char *_Nonnull uuid,
                             CBDriverCharacteristicMapEntry *_Nonnull entries, int entriesLength,
                             char *_Nullable *_Nullable errorOut);
BOOL v23_cbdriver_writeData(const char *_Nonnull data,
                             const int dataLength, char *_Nullable *_Nullable errorOut);
int v23_cbdriver_advertisingServiceCount();
void v23_cbdriver_removeService(const char *_Nonnull uuid);
void v23_cbdriver_setAdRotateDelay(float seconds);

// Discovery via CGO
BOOL v23_cbdriver_startScan(const char *_Nonnull *_Nonnull uuids, int uuidsLength,
                            const char *_Nonnull baseUuid, const char *_Nonnull maskUuid,
                            char *_Nullable *_Nullable errorOut);
void v23_cbdriver_stopScan();

// Removes the driver from memory and stops all bluetooth activity
void v23_cbdriver_clean();
