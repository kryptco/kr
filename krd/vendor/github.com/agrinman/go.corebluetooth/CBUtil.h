// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <CoreBluetooth/CoreBluetooth.h>
#import <Foundation/Foundation.h>

/** Utility function that performs dispatch_sync in a deadlock-resistant fashion
 by checking
 if we're currently on that queue (via same labels), and if so then we just run
 the block now.
 */
void CBDispatchSync(dispatch_queue_t _Nonnull queue, dispatch_block_t _Nonnull block);

@interface CBMutableService (UsefulConstructor)

+ (CBMutableService *_Nonnull)cb_mutableService:(CBUUID *_Nonnull)uuid
                    withReadOnlyCharacteristics:
                        (NSDictionary<CBUUID *, NSData *> *_Nonnull)characteristics;

@end