## LE Command Requirements

List of the commands and events that a Controller supporting LE shall implement.  [Vol 2 Part A.3.19]

- Mandatory

  - [ ] Vol 2, Part E, 7.7.14 - Command Complete Event (0x0E)
  - [ ] Vol 2, Part E, 7.7.15 - Command Status Event (0x0F)
  - [ ] Vol 2, Part E, 7.8.16 - LE Add Device To White List Command (0x08|0x0011)
  - [ ] Vol 2, Part E, 7.8.15 - LE Clear White List Command (0x08|0x0010)
  - [ ] Vol 2, Part E, 7.8.2 - LE Read Buffer Size Command (0x08|0x0002)
  - [ ] Vol 2, Part E, 7.4.3 - Read Local Supported Features Command (0x04|0x0003)
  - [ ] Vol 2, Part E, 7.8.27 - LE Read Supported States Command (0x08|0x001C)
  - [ ] Vol 2, Part E, 7.8.14 - LE Read White List Size Command (0x08|0x000F)
  - [ ] Vol 2, Part E, 7.8.17 - LE Remove Device From White List Command (0x08|0x0012)
  - [ ] Vol 2, Part E, 7.8.1 - LE Set Event Mask Command (0x08|0x0001)
  - [ ] Vol 2, Part E, 7.8.30 - LE Test End Command (0x08|0x001F)
  - [ ] Vol 2, Part E, 7.4.6 - Read BD_ADDR Command (0x04|0x0009)
  - [ ] Vol 2, Part E, 7.8.3 - LE Read Local Supported Features Command (0x08|0x0003)
  - [ ] Vol 2, Part E, 7.4.1 - Read Local Version Information Command (0x04|0x0001)
  - [ ] Vol 2, Part E, 7.3.2 - Reset Command (0x03|0x003)
  - [ ] Vol 2, Part E, 7.4.2 - Read Local Supported Commands Command (0x04|0x0002)
  - [ ] Vol 2, Part E, 7.3.1 - Set Event Mask Command (0x03|0x0001)


- C1: Mandatory if Controller supports transmitting packets, otherwise optional.

  - [ ] Vol 2, Part E, 7.8.6 - LE Read Advertising Channel Tx Power Command (0x08|0x0007)
  - [ ] Vol 2, Part E, 7.8.29 - LE Transmitter Test Command (0x08|0x001E)
  - [ ] Vol 2, Part E, 7.8.9 - LE Set Advertise Enable Command (0x08|0x000A)
  - [ ] Vol 2, Part E, 7.8.7 - LE Set Advertising Data Command (0x08|0x0008)
  - [ ] Vol 2, Part E, 7.8.5 - LE Set Advertising Parameters Command (0x08|0x0006)
  - [ ] Vol 2, Part E, 7.8.4 - LE Set Random Address Command (0x08|0x0005)


- C2: Mandatory if Controller supports receiving packets, otherwise optional.

  - [ ] Vol 2, Part E, 7.7.65.2 - LE Advertising Report Event (0x3E)
  - [ ] Vol 2, Part E, 7.8.28 - LE Receiver Test Command (0x08|0x001D)
  - [ ] Vol 2, Part E, 7.8.11 - LE Set Scan Enable Command (0x08|0x000C)
  - [ ] Vol 2, Part E, 7.8.10 - LE Set Scan Parameters Command (0x08|0x000B)


- C3: Mandatory if Controller supports transmitting and receiving packets, otherwise optional.

  - [ ] Vol 2, Part E, 7.1.6 - Disconnect Command (0x01|0x0006)
  - [ ] Vol 2, Part E, 7.7.5 - Disconnection Complete Event (0x05)
  - [ ] Vol 2, Part E, 7.7.65.1 - LE Connection Complete Event (0x3E)
  - [ ] Vol 2, Part E, 7.8.18 - LE Connection Update Command (0x08|0x0013)
  - [ ] Vol 2, Part E, 7.7.65.3 - LE Connection Update Complete Event (0x0E)
  - [ ] Vol 2, Part E, 7.8.12 - LE Create Connection Command (0x08|0x000D)
  - [ ] Vol 2, Part E, 7.8.13 - LE Create Connection Cancel Command (0x08|0x000E)
  - [ ] Vol 2, Part E, 7.8.20 - LE Read Channel Map Command (0x08|0x0015)
  - [ ] Vol 2, Part E, 7.8.21 - LE Read Remote Used Features Command (0x08|0x0016)
  - [ ] Vol 2, Part E, 7.7.65.4 - LE Read Remote Used Features Complete Event (0x3E)
  - [ ] Vol 2, Part E, 7.8.19 - LE Set Host Channel Classification Command (0x08|0x0014)
  - [ ] Vol 2, Part E, 7.8.8 - LE Set Scan Response Data Command (0x08|0x0009)
  - [ ] Vol 2, Part E, 7.3.40 - Host Number Of Completed Packets Command (0x03|0x0035)
  - [ ] Vol 2, Part E, 7.3.35 - Read Transmit Power Level Command (0x03|0x002D)
  - [ ] Vol 2, Part E, 7.1.23 - Read Remote Version Information Command (0x01|0x001D)
  - [ ] Vol 2, Part E, 7.7.12 - Read Remote Version Information Complete Event (0x0C)
  - [ ] Vol 2, Part E, 7.5.4 - Read RSSI Command (0x05|0x0005)


- C4: Mandatory if LE Feature (LL Encryption) is supported otherwise optional.

  - [ ] Vol 2, Part E, 7.7.8 - Encryption Change Event (0x08)
  - [ ] Vol 2, Part E, 7.7.39 - Encryption Key Refresh Complete Event (0x30)
  - [ ] Vol 2, Part E, 7.8.22 - LE Encrypt Command (0x08|0x0017)
  - [ ] Vol 2, Part E, 7.7.65.5 - LE Long Term Key Request Event (0x3E)
  - [ ] Vol 2, Part E, 7.8.25 - LE Long Term Key Request Reply Command (0x08|0x001A)
  - [ ] Vol 2, Part E, 7.8.26 - LE Long Term Key Request Negative Reply Command (0x08|0x001B)
  - [ ] Vol 2, Part E, 7.8.23 - LE Rand Command (0x08|0x0018)
  - [ ] Vol 2, Part E, 7.8.24 - LE Start Encryption Command (0x08|0x0019)


- C5: Mandatory if BR/EDR is supported otherwise optional. [Won't supported]

  - [ ] Vol 2, Part E, 7.4.5 - Read Buffer Size Command
  - [ ] Vol 2, Part E, 7.3.78 - Read LE Host Support
  - [ ] Vol 2, Part E, 7.3.79 - Write LE Host Support Command (0x03|0x006D)


- C6: Mandatory if LE Feature (Connection Parameters Request procedure) is supported, otherwise optional.

  - [ ] Vol 2, Part E, 7.8.31 - LE Remote Connection Parameter Request Reply Command (0x08|0x0020)
  - [ ] Vol 2, Part E, 7.8.32 - LE Remote Connection Parameter Request Negative Reply Command (0x08|0x0021)
  - [ ] Vol 2, Part E, 7.7.65.6 - LE Remote Connection Parameter Request Event (0x3E)


- C7: Mandatory if LE Ping is supported otherwise excluded

  - [ ] Vol 2, Part E, 7.3.94 - Write Authenticated Payload Timeout Command (0x01|0x007C)
  - [ ] Vol 2, Part E, 7.3.93 - Read Authenticated Payload Timeout Command (0x03|0x007B)
  - [ ] Vol 2, Part E, 7.7.75 - Authenticated Payload Timeout Expired Event (0x57)
  - [ ] Vol 2, Part E, 7.3.69 - Set Event Mask Page 2 Command (0x03|0x0063)


- Optional support

  - [ ] Vol 2, Part E, 7.7.26 - Data Buffer Overflow Event (0x1A)
  - [ ] Vol 2, Part E, 7.7.16 - Hardware Error Event (0x10)
  - [ ] Vol 2, Part E, 7.3.39 - Host Buffer Size Command (0x03|0x0033)
  - [ ] Vol 2, Part E, 7.7.19 - Number Of Completed Packets Event (0x13)
  - [ ] Vol 2, Part E, 7.3.38 - Set Controller To Host Flow Control Command

  ##  Vol 3, Part A, 4 L2CAP Signaling mandatory for LE-U

  - [ ] Vol 3, Part A, 4.1 - Command Reject (0x01)
  - [ ] Vol 3, Part A, 4.6 - Disconnect Request (0x06)
  - [ ] Vol 3, Part A, 4.7 - Disconnect Response (0x07)
  - [ ] Vol 3, Part A, 4.20 - Connection Parameter Update Request (0x12)
  - [ ] Vol 3, Part A, 4.21 - Connection Parameter Update Response (0x13)
  - [ ] Vol 3, Part A, 4.22 - LE Credit Based Connection Request (0x14)
  - [ ] Vol 3, Part A, 4.23 - LE Credit Based Connection Response (0x15)
  - [ ] Vol 3, Part A, 4.24 - LE Flow Control Credit (0x16)
