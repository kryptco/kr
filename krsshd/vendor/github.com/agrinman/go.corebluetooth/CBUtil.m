// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import "CBLog.h"
#import "CBUtil.h"

void CBDispatchSync(dispatch_queue_t _Nonnull queue, dispatch_block_t _Nonnull block) {
  if (strcmp(dispatch_queue_get_label(DISPATCH_CURRENT_QUEUE_LABEL),
             dispatch_queue_get_label(queue)) == 0) {
    // We're on that queue now -- if we did dispatch_sync we'd deadlock so just
    // run it.
    block();
  } else {
    dispatch_sync(queue, block);
  }
}

@implementation CBMutableService (UsefulConstructor)

+ (CBMutableService *_Nonnull)cb_mutableService:(CBUUID *_Nonnull)uuid
                    withReadOnlyCharacteristics:
                        (NSDictionary<CBUUID *, NSData *> *_Nonnull)characteristics {
  CBMutableService *service = [[CBMutableService alloc] initWithType:uuid primary:YES];
  if (characteristics.count > 0) {
    // Create CBCharacteristics from our map
    NSMutableArray<CBMutableCharacteristic *> *cbChars = [NSMutableArray new];
    for (CBUUID *uuid in characteristics) {
      // We don't pass the data here because it would cause it to aggressively
      // cache the data
      // and we want to be able to change it at will.
      CBMutableCharacteristic *c =
          [[CBMutableCharacteristic alloc] initWithType:uuid
                                             properties:CBCharacteristicPropertyRead
                                                  value:nil
                                            permissions:CBAttributePermissionsReadable];
      [cbChars addObject:c];
    }
    CBDebugLog(@"Adding %d characteristics to service %@", cbChars.count, uuid);
    service.characteristics = cbChars;
  }
  return service;
}

@end
