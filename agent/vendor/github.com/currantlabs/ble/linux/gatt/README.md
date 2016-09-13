## Generic Attribute Profile (GATT)

This package implement Generic Attribute Profile (GATT) [Vol 3, Part G]

### Check list for ATT Client implementation.

#### Server Configuration [4.3]
  - [x] Exchange MTU [4.3.1]

#### Primary Service Discovery [4.4]
  - [x] Discover All Primary Service [4.4.1]
  - [ ] Discover Primary Service by Service UUID [4.4.2]

#### Relationship Discovery [4.5]
  - [ ] Find Included Services [4.5.1]

#### Characteristic Discovery [4.6]
  - [x] Discover All Characteristics of a Service [4.6.1]
  - [ ] Discover Characteristics by UUID [4.6.2]

#### Characteristic Descriptors Discovery [4.7]
  - [x] Discover All Characteristic Descriptors [4.7.1]

#### Characteristic Value Read [4.8]
  - [ ] Read Characteristic Value [4.8.1]
  - [ ] Read Using Characteristic UUID [4.8.2]
  - [ ] Read Long Characteristic Values [4.8.3]
  - [ ] Read Multiple Characteristic Values [4.8.4]

#### Characteristic Value Write [4.9]
  - [x] Write Without Response [4.9.1]
  - [ ] Signed Write Without Response [4.9.2]
  - [x] Write Characteristic Value [4.9.3]
  - [ ] Write Long Characteristic Values [4.9.4]
  - [x] Reliable Writes [4.9.5]

#### Characteristic Value Notifications [4.10]
  - [x] Notifications [4.10.1]

#### Characteristic Indications [4.11]
  - [x] Indications [4.11.1]

#### Characteristic Descriptors [4.12]
  - [ ] Read Characteristic Descriptors [4.12.1]
  - [ ] Read Long Characteristic Descriptors [4.12.2]
  - [ ] Write Characteristic Descriptors [4.12.3]
  - [ ] Write Long Characteristic Descriptors [4.12.4]
