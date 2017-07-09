// +build darwin,cgo
// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <Foundation/Foundation.h>

typedef enum { CBLogLevelDebug, CBLogLevelInfo, CBLogLevelError } CBLogLevel;

#define CBLOG_LEVEL_DEBUG 0
#define CBLOG_LEVEL_INFO 1
#define CBLOG_LEVEL_ERROR 2

// This is normally defined already in corebluetooth.go's import "C" section under CFLAGS.
#ifndef CBLOG_LEVEL
#define CBLOG_LEVEL CBLOG_LEVEL_DEBUG
#endif

void _CBLog(CBLogLevel level, const char *_Nonnull file, int line, NSString *_Nonnull format, ...);

#define CBDebugLog(format, args...) _CBLog(CBLogLevelDebug, __FILE__, __LINE__, format, ##args)

#define CBInfoLog(format, args...) _CBLog(CBLogLevelInfo, __FILE__, __LINE__, format, ##args)

#define CBErrorLog(format, args...) _CBLog(CBLogLevelError, __FILE__, __LINE__, format, ##args)
