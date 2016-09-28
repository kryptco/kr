// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import "CBLog.h"

// Exists in GO
extern void v23_corebluetooth_go_log(char *msg);
extern void v23_corebluetooth_go_log_error(char *msg);

void _CBLog(CBLogLevel level, const char *_Nonnull file, int line, NSString *_Nonnull format, ...) {
  va_list args;
  va_start(args, format);
  NSString *rendered = [[NSString alloc] initWithFormat:format arguments:args];
  va_end(args);
  NSString *fileOnly = [[[NSString alloc] initWithUTF8String:file] lastPathComponent];
  rendered = [NSString stringWithFormat:@"%@:%d %@", fileOnly, line, rendered];
  const char *cString = [rendered UTF8String];
  if (cString) {
    switch (level) {
      case CBLogLevelDebug:
      case CBLogLevelInfo:
        v23_corebluetooth_go_log((char *)cString);
        break;
      case CBLogLevelError:
        v23_corebluetooth_go_log_error((char *)cString);
        break;
    }
  } else {
    NSLog(@"Unable to get UTF8String to log to go: %@", rendered);
  }
}
