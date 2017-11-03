//
//  Log.swift
//  Kryptonite
//
//  Created by Alex Grinman on 8/31/16.
//  Copyright © 2016 KryptCo, Inc. Inc. All rights reserved.
//

import Foundation

enum LogType {
    case info
    case warning
    case error
    
    func getIndicator() -> String {
        switch self {
        case .info:
            return "🔵"
        case .warning:
            return "⚠️"
        case .error:
            return "🔴"
        }
    }
}



func log(_ arg:CustomDebugStringConvertible?, _ type:LogType = .info, file:String = #file, function:String = #function, line:Int = #line) {
    let className = URL(fileURLWithPath: file).lastPathComponent.replacingOccurrences(of: ".swift", with: "")
    let statement = "[\(Date().timeIntervalSince1970)] - \(type.getIndicator()) \(className).\(function):\(line)> \(arg ?? "")"
    #if DEBUG
        print(statement)
    #endif
}
