//
//  Bluetooth.swift
//  Kryptonite
//
//  Created by Kevin King on 9/12/16.
//  Copyright © 2016 KryptCo, Inc. All rights reserved.
//

import Foundation
import CoreBluetooth

let globalMutex = Mutex()
var krbtle_on_bluetooth_data : KRBTLE_ON_BLUETOOTH_DATA_T? = nil

@_silgen_name("krbtle_set_on_bluetooth_data")
func krbtle_set_on_bluetooth_data(callback: @escaping KRBTLE_ON_BLUETOOTH_DATA_T) {
    globalMutex.lock {
        krbtle_on_bluetooth_data = callback
    }
}

class Init {
    static let bluetoothManager: BluetoothManager = {
        return BluetoothManager()
    }()
}

func uuidFromUnsafePointer(ptr: UnsafePointer<UInt8>, len: UInt64) -> CBUUID? {
    let uuidData = Data(bytes: ptr, count: Int(len))
    guard let string = String(bytes: uuidData, encoding: .utf8) else {
        log("invalid uuid string")
        return nil
    }
    guard let uuid = UUID(uuidString: string) else {
        return nil
    }
    return CBUUID(nsuuid: uuid)
}

@_silgen_name("krbtle_add_service")
func addService(serviceUUIDPtr: UnsafePointer<UInt8>, serviceUUIDLen: UInt64) {
    log("addService")
    guard let serviceUUID = uuidFromUnsafePointer(ptr: serviceUUIDPtr, len: serviceUUIDLen) else {
        return
    }
    Init.bluetoothManager.add(session: serviceUUID)
}

@_silgen_name("krbtle_remove_service")
func removeService(serviceUUIDPtr: UnsafePointer<UInt8>, serviceUUIDLen: UInt64) {
    log("removeService")
    guard let serviceUUID = uuidFromUnsafePointer(ptr: serviceUUIDPtr, len: serviceUUIDLen) else {
        return
    }
    Init.bluetoothManager.remove(session: serviceUUID)
}

@_silgen_name("krbtle_write_data")
func writeData(serviceUUIDPtr: UnsafePointer<UInt8>, serviceUUIDLen: UInt64,
                dataPtr: UnsafePointer<UInt8>, dataLen: UInt64) {
    let data = Data(bytes: dataPtr, count: Int(dataLen))
    guard let msg = try? NetworkMessage(networkData: data) else {
        log("bad msg")
        return
    }
    let serviceUUIDData = Data(bytes: serviceUUIDPtr, count: Int(serviceUUIDLen))
    guard let serviceUUIDString = String(bytes: serviceUUIDData, encoding: .utf8) else {
        log("invalid uuid string")
        return
    }
    let serviceUUID = CBUUID(string: serviceUUIDString)
    Init.bluetoothManager.send(message: msg, for: serviceUUID, completionHandler: nil)
}

@_silgen_name("krbtle_stop")
func krbtle_stop() {
    Init.bluetoothManager.removeAll()
}

class BluetoothManager {
    
    var centralManager:CBCentralManager
    var bluetoothDelegate:BluetoothDelegate
    var mutex = Mutex()
    let queue = DispatchQueue.global()
    
    required init() {
        self.bluetoothDelegate = BluetoothDelegate(queue: queue)
        self.centralManager = CBCentralManager(delegate: bluetoothDelegate, queue: queue)
        self.bluetoothDelegate.onReceive = onBluetoothReceive
        self.bluetoothDelegate.central = self.centralManager
        log("initialized")
    }

    //MARK: Transport
    
    func send(message:NetworkMessage, for session:CBUUID, completionHandler: (()->Void)?) {
        //TODO: bluetooth completion
        queue.async {
            self.bluetoothDelegate.writeToServiceUUID(uuid: session, message: message)
        }

    }
    func add(session:CBUUID) {
        mutex.lock {
            let uuid = session
            bluetoothDelegate.addServiceUUID(uuid: uuid)
        }
    }
    func remove(session:CBUUID) {
        mutex.lock {
            bluetoothDelegate.removeServiceUUID(uuid: session)
        }
    }
    func removeAll() {
        mutex.lock {
            let allUUIDS = bluetoothDelegate.allServiceUUIDS
            for uuid in allUUIDS {
                bluetoothDelegate.removeServiceUUID(uuid: uuid)
            }
        }
    }
    func willEnterBackground() {
        // do nothing
    }
    func willEnterForeground() {
        // do nothing
    }

    func refresh(for session:CBUUID) {
        bluetoothDelegate.refreshServiceUUID(uuid: session)
    }

    // MARK: Bluetooth
    func onBluetoothReceive(serviceUUID: CBUUID, message: NetworkMessage) throws {
        mutex.lock()
        var data = message.networkFormat()
        data.withUnsafeBytes { (u8Ptr: UnsafePointer<UInt8>) in
            globalMutex.lock {
                if let callback = krbtle_on_bluetooth_data {
                    callback(u8Ptr, UInt64(data.count))
                }
            }
        }
        mutex.unlock()
    }

}


typealias BluetoothOnReceiveCallback = (CBUUID, NetworkMessage) throws -> Void

class BluetoothDelegate : NSObject, CBCentralManagerDelegate, CBPeripheralDelegate {
    
    var central:CBCentralManager?
    var onReceive:BluetoothOnReceiveCallback?
    
    var allServiceUUIDS : Set<CBUUID> = Set()
    var scanningServiceUUIDS: Set<CBUUID> = Set()
    var pairedServiceUUIDS: Set<CBUUID> = Set()
    var startFullScanAfterUnixSeconds : Int64 = 0
    var scanStartedUnixSeconds : Int64? = nil
    var scanEpoch : UInt64 = 0

    //  reconnect to old known peripheral UUID even if it's not advertising services
    var cachedPeripheralUUIDS : [CBUUID] = []

    //  detect unresponsive peripheral
    var lastReadUnixSeconds : Int64? = nil
    var refreshEpoch : UInt64 = 0

    var discoveredPeripherals : Set<CBPeripheral> = Set()
    var pairedPeripherals: [CBUUID: CBPeripheral] = [:]
    var peripheralCharacteristics: [CBPeripheral: CBCharacteristic] = [:]

    var characteristicMessageBuffersAndLastSplitNumber: [CBCharacteristic: (Data, UInt8)] = [:]
    var lastOutgoingServiceUUIDAndMessage : (CBUUID, NetworkMessage)? = nil

    var mutex : Mutex = Mutex()
    let queue: DispatchQueue
    
    // Constants
    static let krsshCharUUID = CBUUID(string: "20F53E48-C08D-423A-B2C2-1C797889AF24")
    static let refreshByte = UInt8(0)
    static let pingByte = UInt8(1)
    static let pingMsg = Data(bytes: [pingByte])

    init(queue: DispatchQueue) {
        self.queue = queue
        super.init()
        log("init bluetooth")
    }

    func centralManagerDidUpdateState(_ central: CBCentralManager) {
        mutex.lock()
        defer { mutex.unlock() }
        log("CBCentral state \(central.state.rawValue)")
        if central.state == .poweredOn {
            self.central = central
            scanStartedUnixSeconds = nowUnixSeconds()
            log("CBCentral poweredOn")
            for peripheral in discoveredPeripherals {
                restorePeripheralLocked(central, peripheral)
            }
        }
        scanLogic(scanEpoch)
    }

    //  re-initialize a peripheral being restored from background or from Bluetooth being toggled back on
    func restorePeripheralLocked(_ central : CBCentralManager, _ peripheral: CBPeripheral) {
        log("restoring peripheral \(peripheral)")
        switch peripheral.state {
        case .disconnected, .disconnecting:
            removePeripheralLocked(central: central, peripheral: peripheral)
            break
        case .connecting:
            //  peripherals in the connecting state will likely not finish connecting
            //  and persist in this bad state across app launches, so cancel any that are still
            //  connecting on poweredOn state transition
            discoveredPeripherals.insert(peripheral)
            if case .poweredOn = central.state {
                log("cancelling connecting discoveredPeripheral on poweredOn: \(peripheral)")
                central.cancelPeripheralConnection(peripheral)
            }
        case .connected:
            discoveredPeripherals.insert(peripheral)
            if case .poweredOn = central.state {
                peripheral.discoverServices(Array(self.scanningServiceUUIDS))
            }
        }
    }

    func writeToServiceUUID(uuid: CBUUID, message: NetworkMessage) {
        mutex.lock()
        defer { mutex.unlock() }
        lastOutgoingServiceUUIDAndMessage = (uuid, message)
        let data = message.networkFormat()
        scanStartedUnixSeconds = nowUnixSeconds()
        
        guard let peripheral = pairedPeripherals[uuid],
            let characteristic = peripheralCharacteristics[peripheral] else {
            return
        }
        do {
            let mtu = UInt(peripheral.maximumWriteValueLength(for: .withResponse))
            log("mtu: \(mtu)")
            let messageBlocks = try splitMessageForBluetooth(message: data, mtu: mtu)
            for block in messageBlocks {
                peripheral.writeValue(block, for: characteristic, type: .withResponse)
                let lastWrite = nowUnixSeconds()
                let currentRefreshEpoch = self.refreshEpoch
                queue.asyncAfter(delay: 4, task: {
                    self.mutex.lock()
                    defer { self.mutex.unlock() }
                    guard let lastRead = self.lastReadUnixSeconds,
                        lastRead >= lastWrite else {
                            if currentRefreshEpoch == self.refreshEpoch {
                                log("refreshing due to inactivity")
                                self.refreshEpoch += 1
                                self.refreshServiceUUIDLocked(uuid: uuid)
                            }
                        return
                    }
                })
                log("sent BT packet")
            }
        } catch let e {
            log("bluetooth message split failed: \(e)", .error)
        }
    }

    func writeToServiceUUIDRawLockedAcked(uuid: CBUUID, data: Data) {
        guard let peripheral = pairedPeripherals[uuid],
            let characteristic = peripheralCharacteristics[peripheral] else {
                return
        }
        peripheral.writeValue(data, for: characteristic, type: .withResponse)
    }

    func scanLogic(_ epoch: UInt64) {
        guard let central = central else {
            log("central nil")
            return
        }

        guard epoch == scanEpoch else {
            return
        }
        scanEpoch += 1

        if central.state != .poweredOn {
            scanningServiceUUIDS.removeAll()
            pairedServiceUUIDS.removeAll()
            log("not poweredOn: \(central.state.rawValue)")
            return
        }

        let shouldBeScanning = allServiceUUIDS.subtracting(pairedServiceUUIDS)
        if shouldBeScanning.count == 0 {
            log("Stop scanning")
            if #available(OSX 10.13, *) {
                if central.isScanning {
                    central.stopScan()
                }
            } else {
                // Fallback on earlier versions
                central.stopScan()
            }
            return
        }

        scanningServiceUUIDS = shouldBeScanning

        //  Scan until 20 seconds after `fullScanStartedUnixSeconds`
        //  Scan for the cached peripheral UUIDs after `startFullScanAfterUnixSeconds` and when cache is non-empty,
        //  else scan for the specific service (which only works when iOS is in foreground).
        //  Otherwise, stop scanning to conserve power.
        let now = nowUnixSeconds()
        if let scanStartedUnixSeconds = scanStartedUnixSeconds,
            now - scanStartedUnixSeconds <= 20 {
            if now >= startFullScanAfterUnixSeconds && cachedPeripheralUUIDS.count > 0 {
                //  continue full scan for at least 20 seconds
                log("Scanning all devices")
                central.scanForPeripherals(withServices: nil, options: [CBCentralManagerScanOptionAllowDuplicatesKey: true])
            } else {
                log("Scanning for \(scanningServiceUUIDS)")
                central.scanForPeripherals(withServices: Array(scanningServiceUUIDS), options: [CBCentralManagerScanOptionAllowDuplicatesKey: true])
            }
        } else {
            log("Stop scanning")
            central.stopScan()
        }
        let epoch = scanEpoch
        queue.asyncAfter(delay: 5, task: {
            self.scanLogic(epoch)
        })
    }

    func addServiceUUID(uuid: CBUUID) {
        mutex.lock()
        defer { mutex.unlock() }

        if allServiceUUIDS.contains(uuid) {
             log("already had uuid \(uuid.uuidString)")
            return
        }

        do {
            cachedPeripheralUUIDS = try loadStoredPeripheralUUIDS(for: uuid)
        } catch {}

        log("add uuid \(uuid.uuidString)")
        allServiceUUIDS.insert(uuid)
        //  since user might be pairing, iOS app could be in foreground, so try explicitly scanning
        queue.async {
            self.scanStartedUnixSeconds = nowUnixSeconds()
            self.startFullScanAfterUnixSeconds = nowUnixSeconds() + 3
            self.scanLogic(self.scanEpoch)
        }
    }

    func removeServiceUUID(uuid: CBUUID) {
        mutex.lock()
        defer { mutex.unlock() }
        removeServiceUUIDLocked(uuid: uuid)
    }

    func removeServiceUUIDLocked(uuid: CBUUID) {
        if !allServiceUUIDS.contains(uuid) {
            log("didn't have uuid \(uuid.uuidString) in allServiceUUIDS")
        } else {
            log("remove uuid \(uuid.uuidString)")
        }
        allServiceUUIDS.remove(uuid)
        pairedServiceUUIDS.remove(uuid)
        scanningServiceUUIDS.remove(uuid)
        if let pairedPeripheral = pairedPeripherals.removeValue(forKey: uuid) {
            if let characteristic = peripheralCharacteristics.removeValue(forKey: pairedPeripheral) {
                if characteristic.isNotifying {
                    pairedPeripheral.setNotifyValue(false, for: characteristic)
                }
            }
            //  Check if any remaining serviceUUIDs for peripheral
            if !pairedPeripherals.values.contains(pairedPeripheral) {
                central?.cancelPeripheralConnection(pairedPeripheral)
            }
        }
        scanLogic(scanEpoch)
    }

    func refreshServiceUUID(uuid: CBUUID) {
        mutex.lock()
        defer { mutex.unlock() }
        refreshServiceUUIDLocked(uuid: uuid)
    }

    func refreshServiceUUIDLocked(uuid: CBUUID) {
        guard allServiceUUIDS.contains(uuid) else {
            log("not refreshing unknown uuid \(uuid.uuidString)", .error)
            return
        }
        log("refresh uuid \(uuid.uuidString)")
        if let peripheral = pairedPeripherals[uuid] {
            if let characteristic = peripheralCharacteristics[peripheral] {
                if characteristic.isNotifying {
                    peripheral.setNotifyValue(false, for: characteristic)
                }
            }
            central?.cancelPeripheralConnection(peripheral)
        }
        removeServiceUUIDLocked(uuid: uuid)
        allServiceUUIDS.insert(uuid)
        startFullScanAfterUnixSeconds = nowUnixSeconds() + 3
        scanStartedUnixSeconds = nowUnixSeconds()
        scanLogic(scanEpoch)
    }

    func centralManager(_ central: CBCentralManager, didDiscover peripheral: CBPeripheral, advertisementData: [String : Any], rssi RSSI: NSNumber) {
        mutex.lock()
        defer { mutex.unlock() }
        if let name = peripheral.name {
            log("Discovered \(name) \(peripheral) at RSSI \(RSSI)")
        }
        log("\(advertisementData)")
        guard shouldConnectPeripheral(central, peripheral, advertisementData) else {
            return
        }
        peripheral.delegate = self
        //  keep reference so not GCed
        discoveredPeripherals.insert(peripheral)

        connectPeripheral(central, peripheral)
    }

    func shouldConnectPeripheral(_ central: CBCentralManager, _ peripheral: CBPeripheral, _ advertisementData: [String : Any]) -> Bool {
        if let advertisedServices = advertisementData[CBAdvertisementDataServiceUUIDsKey] as? [CBUUID] {
            if advertisedServices.contains(where: {allServiceUUIDS.contains($0)}) {
            log("discovered peripheral with matching service")
                return true
            }
        }
        if let overflowServices = advertisementData[CBAdvertisementDataOverflowServiceUUIDsKey] as? [CBUUID] {
            if overflowServices.contains(where: {allServiceUUIDS.contains($0)}) {
                log("discovered peripheral with matching overflow service")
                return true
            }
        }
        if let newPeripheralUUID = getPeripheralIdentifierHack(peripheral),
            cachedPeripheralUUIDS.contains(newPeripheralUUID) {
            log("discovered cached peripheral")
            return true
        }
        return false
    }

    func connectPeripheral(_ central: CBCentralManager, _ peripheral: CBPeripheral) {
        guard central.state == .poweredOn else {
            return
        }

        central.connect(peripheral, options: nil)
    }

    func centralManager(_ central: CBCentralManager, didConnect peripheral: CBPeripheral) {
        mutex.lock()
        defer { mutex.unlock() }
        if #available(OSX 10.13, *) {
            log("connected \(peripheral.identifier)")
        } else {
            // Fallback on earlier versions
            log("connected \(peripheral)")
        }
        peripheral.delegate = self
        peripheral.discoverServices(Array(allServiceUUIDS))
    }

    func centralManager(_ central: CBCentralManager, didFailToConnect peripheral: CBPeripheral, error: Error?) {
        mutex.lock()
        defer { mutex.unlock() }
        if #available(OSX 10.13, *) {
            log("failed to connect \(peripheral.identifier)")
        } else {
            // Fallback on earlier versions
            log("failed to connect \(peripheral)")
        }
        discoveredPeripherals.remove(peripheral)
        removePeripheralLocked(central: central, peripheral: peripheral)
    }

    func peripheral(_ peripheral: CBPeripheral, didDiscoverServices error: Error?) {
        mutex.lock()
        defer { mutex.unlock() }
        guard let services = peripheral.services else {
            return
        }
        var foundPairedServiceUUID = false
        log("discovered services \(services)")
        for service in services {
            guard allServiceUUIDS.contains(service.uuid) else {
                continue
            }
            if let peripheralUUID = getPeripheralIdentifierHack(peripheral) {
                try? savePeripheralUUID(service: service.uuid, peripheral: peripheralUUID)
                cachedPeripheralUUIDS = [peripheralUUID]
            }
            foundPairedServiceUUID = true
            log("discovered service UUID \(service.uuid)")
            pairedPeripherals[service.uuid] = peripheral
            pairedServiceUUIDS.insert(service.uuid)
            scanningServiceUUIDS.remove(service.uuid)
            peripheral.discoverCharacteristics([BluetoothDelegate.krsshCharUUID], for: service)
        }
        if !foundPairedServiceUUID {
            if #available(OSX 10.13, *) {
                log("disconnected peripheral with no relevant services \(peripheral.identifier.uuidString)")
            } else {
                // Fallback on earlier versions
                log("disconnected peripheral with no relevant services \(peripheral)")
            }
            central?.cancelPeripheralConnection(peripheral)
            if let peripheralUUID = getPeripheralIdentifierHack(peripheral) {
                cachedPeripheralUUIDS = cachedPeripheralUUIDS.filter({$0 != peripheralUUID })
            }
        }
        scanLogic(scanEpoch)
    }

    func peripheral(_ peripheral: CBPeripheral, didModifyServices invalidatedServices: [CBService]) {
        log("services invalidated, refreshing")
        //  prevent connecting too early to cached services
        for service in invalidatedServices {
            let uuid = service.uuid
            queue.asyncAfter(delay: 1, task: {
                self.refreshServiceUUID(uuid: uuid)
            })
        }
    }

    func peripheral(_ peripheral: CBPeripheral,
                    didDiscoverCharacteristicsFor service: CBService,
                    error: Error?){
        mutex.lock()
        defer { mutex.unlock() }

        guard let chars = service.characteristics else {
            log("service has no characteristics")
            return
        }

        guard allServiceUUIDS.contains(service.uuid) else {
            log("found characterisics for unpaired service \(service.uuid)")
            return
        }
        for char in chars {
            guard char.uuid.isEqual(BluetoothDelegate.krsshCharUUID) else {
                log("found non-krSSH characteristic \(char.uuid)")
                continue
            }
            log("discovered krSSH characteristic")
            peripheralCharacteristics[peripheral] = char
            if !char.isNotifying {
                peripheral.setNotifyValue(true, for: char)
            }
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didUpdateNotificationStateFor characteristic: CBCharacteristic, error: Error?) {
        mutex.lock()
        defer { mutex.unlock() }
        if let error = error {
            log("Error changing notification state: \(error.localizedDescription)", .error)
        }

        guard characteristic.uuid.isEqual(BluetoothDelegate.krsshCharUUID) else {
            return
        }

        let service = characteristic.service
        if (characteristic.isNotifying) {
            log("Notification began on \(characteristic)")
            if let (uuid, message) = lastOutgoingServiceUUIDAndMessage,
                uuid == service.uuid {
                queue.async {
                    self.writeToServiceUUID(uuid: service.uuid, message: message)
                }
            }
        } else {
            log("Notification stopped on (\(characteristic))  Disconnecting")
            central?.cancelPeripheralConnection(peripheral)
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didUpdateValueFor characteristic: CBCharacteristic, error: Error?) {
        if let error = error {
            log("Error reading characteristic: \(error)", .error)
            return
        }

        guard let data = characteristic.value else {
            return
        }

        if data.count == 1 {
            //  control messages
            switch data[0] {
            case BluetoothDelegate.refreshByte:
                let uuid = characteristic.service.uuid
                log("received refresh control message")
                dispatchAfter(delay: 5.0, task: { self.queue.async{ self.refreshServiceUUID(uuid: uuid) } })
            case BluetoothDelegate.pingByte:
                break
            default:
                break
            }
            return
        }

        mutex.lock {
            lastReadUnixSeconds = nowUnixSeconds()
            onUpdateCharacteristicValue(characteristic: characteristic, data: data)
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didWriteValueFor descriptor: CBDescriptor, error: Error?) {
        if let e = error {
            log("descriptor write error \(e)", .error)
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didWriteValueFor characteristic: CBCharacteristic, error: Error?) {
        if let e = error {
            log("characteristic write error \(e)", .error)
            refreshServiceUUID(uuid: characteristic.service.uuid)
        } else {
            log("characteristic write success")
        }
    }

    func peripheral(_ peripheral: CBPeripheral, didUpdateValueFor descriptor: CBDescriptor, error: Error?) {
        if let e = error {
            log("\(descriptor) error: \(e)")
        }
    }

    //  precondition: mutex locked
    func onUpdateCharacteristicValue(characteristic: CBCharacteristic, data: Data) {
        if data.count == 0 {
            return
        }

        let n = data[0]
        log("received split \(n)")
        let data = data.subdata(in: 1..<data.count)
        
        let bufferAndLastN = characteristicMessageBuffersAndLastSplitNumber[characteristic]
        
        if var buffer = bufferAndLastN?.0, let lastN = bufferAndLastN?.1, lastN > 0, (n == lastN - 1) {
            buffer.append(data)
            characteristicMessageBuffersAndLastSplitNumber[characteristic] = (buffer, n)
        } else {
            characteristicMessageBuffersAndLastSplitNumber[characteristic] = (data, n)
        }
        
        if n == 0 {
            // buffer complete
            if let (fullBuffer, _) = characteristicMessageBuffersAndLastSplitNumber.removeValue(forKey: characteristic) {
                log("reconstructed full message of length \(fullBuffer.count)")
                
                mutex.unlock()
                do {
                    let message = try NetworkMessage(networkData: fullBuffer)
                    try self.onReceive?(characteristic.service.uuid, message)
                } catch (let e) {
                    log("error processing bluetooth message: \(e)")
                }
                mutex.lock()

            }
        }
    }

    func centralManager(_ central: CBCentralManager, didDisconnectPeripheral peripheral: CBPeripheral, error: Error?) {
        mutex.lock()
        defer { mutex.unlock() }
        //  Without destructuring the optional error, the objc runtime crashes from double-releasing it
        if let error = error {
            if #available(OSX 10.13, *) {
                log("Peripheral \(peripheral.identifier) disconnected, error \(error))")
            } else {
                // Fallback on earlier versions
                log("Peripheral \(peripheral) disconnected, error \(error)")
            }
        }

        removePeripheralLocked(central: central, peripheral: peripheral)

        scanLogic(scanEpoch)
    }

    func removePeripheralLocked(central: CBCentralManager, peripheral: CBPeripheral) {
        for disconnectedUUID in pairedPeripherals.filter({ $0.1 == peripheral }).map({$0.0}) {
            log("service uuid disconnected \(disconnectedUUID)")
            pairedPeripherals.removeValue(forKey: disconnectedUUID)
            pairedServiceUUIDS.remove(disconnectedUUID)
            startFullScanAfterUnixSeconds = nowUnixSeconds()
        }
        discoveredPeripherals.remove(peripheral)
    }


}

struct BluetoothMessageTooLong : Error {
    var messageLength : Int
    var mtu : Int
}

/*
 *  Bluetooth packet protocol: 
 *  1-byte message = control message:
 *      0: disconnect from workstation
 *      1: ping/pong
 *  multi-byte message = data
 *      first byte of message indicates number of
 *      remaining packets. Packets must be <= mtu in length.
 */
func splitMessageForBluetooth(message: Data, mtu: UInt) throws -> [Data] {
    let msgBlockSize = mtu - 1;
    let (intN, overflow) = UInt(message.count).dividedReportingOverflow(by: msgBlockSize)
    if overflow || intN > 255 {
        throw BluetoothMessageTooLong(messageLength: message.count, mtu: Int(mtu))
    }
    var blocks : [Data] = []
    let n: UInt8 = UInt8(intN)
    var offset = Int(0)
    for n in (0...n).reversed() {
        var block = Data()
        var inoutN = n
        block.append(&inoutN, count: 1)
        let endIndex = min(message.count, offset + Int(msgBlockSize))
        block.append(message.subdata(in: offset..<endIndex))

        blocks.append(block)
        offset += Int(msgBlockSize)
    }
    return blocks
}

func getPeripheralIdentifierHack(_ peripheral: CBPeripheral) -> CBUUID? {
    if #available(OSX 10.13, *) {
        return CBUUID(nsuuid: peripheral.identifier)
    }
    guard let uuidString = "\(peripheral)".components(separatedBy: "identifier = ").filter({ $0.count > 0 }).dropFirst().first?.components(separatedBy: ",").filter({ $0.count > 0 }).first,
        let uuid = UUID(uuidString: uuidString) else {
        return nil
    }
    return CBUUID(nsuuid: uuid)
}

func getPeripheralCachePath() -> String {
    let homeDir = URL(fileURLWithPath: NSHomeDirectory()).path
    let cachePath = homeDir + "/.kr/bt_peripheral_cache"
    return cachePath
}

func loadStoredPeripheralUUIDS(for service: CBUUID) throws -> [CBUUID] {
    let cachePath = getPeripheralCachePath()
    let cache = try String(contentsOfFile: cachePath)
    let uuids = cache.components(separatedBy: ",").flatMap({UUID(uuidString: $0)}).map({CBUUID(nsuuid: $0)})
    guard uuids.count == 2 else {
        return []
    }
    guard service == uuids[0] else {
        return []
    }
    return [uuids[1]]
}

func savePeripheralUUID(service: CBUUID, peripheral: CBUUID) throws {
    let cache = "\(service.uuidString),\(peripheral.uuidString)"
    let cachePath = getPeripheralCachePath()
    try cache.write(toFile: cachePath, atomically: true, encoding: .utf8)
}
