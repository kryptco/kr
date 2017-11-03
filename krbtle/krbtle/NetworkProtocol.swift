//
//  NetworkProtocol.swift
//  Kryptonite
//
//  Created by Kevin King on 9/25/16.
//  Copyright © 2016 KryptCo, Inc. All rights reserved.
//

import Foundation

struct NetworkMessage {
    struct EncodingError:Error{}

    enum Header:UInt8 {
        case ciphertext = 0x00
        
        // 1.x.x
        case wrappedKey = 0x01
        
        case wrappedPublicKey = 0x02
    }
    
    let header:Header
    let data:Data

    func networkFormat() -> Data {
        var networkData = Data([header.rawValue])
        networkData.append(data)
        return networkData
    }

    init(networkData:Data) throws {
        guard let headerByte = networkData.first,
            let header = Header(rawValue: headerByte) else {
            throw EncodingError()
        }

        self.header = header
        self.data = networkData.subdata(in: 1..<networkData.count)
    }

    init(localData:Data, header:Header) {
        self.data = localData
        self.header = header
    }
}
