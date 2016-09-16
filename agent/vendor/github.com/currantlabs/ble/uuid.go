package ble

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

// A UUID is a BLE UUID.
type UUID []byte

// UUID16 converts a uint16 (such as 0x1800) to a UUID.
func UUID16(i uint16) UUID {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, i)
	return UUID(b)
}

// Parse parses a standard-format UUID string, such
// as "1800" or "34DA3AD1-7110-41A1-B1EF-4430F509CDE7".
func Parse(s string) (UUID, error) {
	s = strings.Replace(s, "-", "", -1)
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if err := lenErr(len(b)); err != nil {
		return nil, err
	}
	return UUID(Reverse(b)), nil
}

// MustParse parses a standard-format UUID string,
// like Parse, but panics in case of error.
func MustParse(s string) UUID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// lenErr returns an error if n is an invalid UUID length.
func lenErr(n int) error {
	switch n {
	case 2, 16:
		return nil
	}
	return fmt.Errorf("UUIDs must have length 2 or 16, got %d", n)
}

// Len returns the length of the UUID, in bytes.
// BLE UUIDs are either 2 or 16 bytes.
func (u UUID) Len() int {
	return len(u)
}

// String hex-encodes a UUID.
func (u UUID) String() string {
	return fmt.Sprintf("%X", Reverse(u))
}

// Equal returns a boolean reporting whether v represent the same UUID as u.
func (u UUID) Equal(v UUID) bool {
	return bytes.Equal(u, v)
}

// Contains returns a boolean reporting whether u is in the slice s.
func Contains(s []UUID, u UUID) bool {
	if s == nil {
		return true
	}

	for _, a := range s {
		if a.Equal(u) {
			return true
		}
	}

	return false
}

// Reverse returns a reversed copy of u.
func Reverse(u []byte) []byte {
	// Special-case 16 bit UUIDS for speed.
	l := len(u)
	if l == 2 {
		return []byte{u[1], u[0]}
	}
	b := make([]byte, l)
	for i := 0; i < l/2+1; i++ {
		b[i], b[l-i-1] = u[l-i-1], u[i]
	}
	return b
}

// Name returns name of know services, characteristics, or descriptors.
func Name(u UUID) string {
	return knownUUID[strings.ToUpper(u.String())].Name
}

// A dictionary of known service names and type (keyed by service uuid)
var knownUUID = map[string]struct{ Name, Type string }{
	"1800": {Name: "Generic Access", Type: "org.bluetooth.service.generic_access"},
	"1801": {Name: "Generic Attribute", Type: "org.bluetooth.service.generic_attribute"},
	"1802": {Name: "Immediate Alert", Type: "org.bluetooth.service.immediate_alert"},
	"1803": {Name: "Link Loss", Type: "org.bluetooth.service.link_loss"},
	"1804": {Name: "Tx Power", Type: "org.bluetooth.service.tx_power"},
	"1805": {Name: "Current Time Service", Type: "org.bluetooth.service.current_time"},
	"1806": {Name: "Reference Time Update Service", Type: "org.bluetooth.service.reference_time_update"},
	"1807": {Name: "Next DST Change Service", Type: "org.bluetooth.service.next_dst_change"},
	"1808": {Name: "Glucose", Type: "org.bluetooth.service.glucose"},
	"1809": {Name: "Health Thermometer", Type: "org.bluetooth.service.health_thermometer"},
	"180A": {Name: "Device Information", Type: "org.bluetooth.service.device_information"},
	"180D": {Name: "Heart Rate", Type: "org.bluetooth.service.heart_rate"},
	"180E": {Name: "Phone Alert Status Service", Type: "org.bluetooth.service.phone_alert_service"},
	"180F": {Name: "Battery Service", Type: "org.bluetooth.service.battery_service"},
	"1810": {Name: "Blood Pressure", Type: "org.bluetooth.service.blood_pressuer"},
	"1811": {Name: "Alert Notification Service", Type: "org.bluetooth.service.alert_notification"},
	"1812": {Name: "Human Interface Device", Type: "org.bluetooth.service.human_interface_device"},
	"1813": {Name: "Scan Parameters", Type: "org.bluetooth.service.scan_parameters"},
	"1814": {Name: "Running Speed and Cadence", Type: "org.bluetooth.service.running_speed_and_cadence"},
	"1815": {Name: "Cycling Speed and Cadence", Type: "org.bluetooth.service.cycling_speed_and_cadence"},

	// A dictionary of known descriptor names and type (keyed by attribute uuid)
	"2800": {Name: "Primary Service", Type: "org.bluetooth.attribute.gatt.primary_service_declaration"},
	"2801": {Name: "Secondary Service", Type: "org.bluetooth.attribute.gatt.secondary_service_declaration"},
	"2802": {Name: "Include", Type: "org.bluetooth.attribute.gatt.include_declaration"},
	"2803": {Name: "Characteristic", Type: "org.bluetooth.attribute.gatt.characteristic_declaration"},

	// A dictionary of known descriptor names and type (keyed by descriptor uuid)
	"2900": {Name: "Characteristic Extended Properties", Type: "org.bluetooth.descriptor.gatt.characteristic_extended_properties"},
	"2901": {Name: "Characteristic User Description", Type: "org.bluetooth.descriptor.gatt.characteristic_user_description"},
	"2902": {Name: "Client Characteristic Configuration", Type: "org.bluetooth.descriptor.gatt.client_characteristic_configuration"},
	"2903": {Name: "Server Characteristic Configuration", Type: "org.bluetooth.descriptor.gatt.server_characteristic_configuration"},
	"2904": {Name: "Characteristic Presentation Format", Type: "org.bluetooth.descriptor.gatt.characteristic_presentation_format"},
	"2905": {Name: "Characteristic Aggregate Format", Type: "org.bluetooth.descriptor.gatt.characteristic_aggregate_format"},
	"2906": {Name: "Valid Range", Type: "org.bluetooth.descriptor.valid_range"},
	"2907": {Name: "External Report Reference", Type: "org.bluetooth.descriptor.external_report_reference"},
	"2908": {Name: "Report Reference", Type: "org.bluetooth.descriptor.report_reference"},

	// A dictionary of known characteristic names and type (keyed by characteristic uuid)
	"2A00": {Name: "Device Name", Type: "org.bluetooth.characteristic.ble.device_name"},
	"2A01": {Name: "Appearance", Type: "org.bluetooth.characteristic.ble.appearance"},
	"2A02": {Name: "Peripheral Privacy Flag", Type: "org.bluetooth.characteristic.ble.peripheral_privacy_flag"},
	"2A03": {Name: "Reconnection Address", Type: "org.bluetooth.characteristic.ble.reconnection_address"},
	"2A04": {Name: "Peripheral Preferred Connection Parameters", Type: "org.bluetooth.characteristic.ble.peripheral_preferred_connection_parameters"},
	"2A05": {Name: "Service Changed", Type: "org.bluetooth.characteristic.gatt.service_changed"},
	"2A06": {Name: "Alert Level", Type: "org.bluetooth.characteristic.alert_level"},
	"2A07": {Name: "Tx Power Level", Type: "org.bluetooth.characteristic.tx_power_level"},
	"2A08": {Name: "Date Time", Type: "org.bluetooth.characteristic.date_time"},
	"2A09": {Name: "Day of Week", Type: "org.bluetooth.characteristic.day_of_week"},
	"2A0A": {Name: "Day Date Time", Type: "org.bluetooth.characteristic.day_date_time"},
	"2A0C": {Name: "Exact Time 256", Type: "org.bluetooth.characteristic.exact_time_256"},
	"2A0D": {Name: "DST Offset", Type: "org.bluetooth.characteristic.dst_offset"},
	"2A0E": {Name: "Time Zone", Type: "org.bluetooth.characteristic.time_zone"},
	"2A0F": {Name: "Local Time Information", Type: "org.bluetooth.characteristic.local_time_information"},
	"2A11": {Name: "Time with DST", Type: "org.bluetooth.characteristic.time_with_dst"},
	"2A12": {Name: "Time Accuracy", Type: "org.bluetooth.characteristic.time_accuracy"},
	"2A13": {Name: "Time Source", Type: "org.bluetooth.characteristic.time_source"},
	"2A14": {Name: "Reference Time Information", Type: "org.bluetooth.characteristic.reference_time_information"},
	"2A16": {Name: "Time Update Control Point", Type: "org.bluetooth.characteristic.time_update_control_point"},
	"2A17": {Name: "Time Update State", Type: "org.bluetooth.characteristic.time_update_state"},
	"2A18": {Name: "Glucose Measurement", Type: "org.bluetooth.characteristic.glucose_measurement"},
	"2A19": {Name: "Battery Level", Type: "org.bluetooth.characteristic.battery_level"},
	"2A1C": {Name: "Temperature Measurement", Type: "org.bluetooth.characteristic.temperature_measurement"},
	"2A1D": {Name: "Temperature Type", Type: "org.bluetooth.characteristic.temperature_type"},
	"2A1E": {Name: "Intermediate Temperature", Type: "org.bluetooth.characteristic.intermediate_temperature"},
	"2A21": {Name: "Measurement Interval", Type: "org.bluetooth.characteristic.measurement_interval"},
	"2A22": {Name: "Boot Keyboard Input Report", Type: "org.bluetooth.characteristic.boot_keyboard_input_report"},
	"2A23": {Name: "System ID", Type: "org.bluetooth.characteristic.system_id"},
	"2A24": {Name: "Model Number String", Type: "org.bluetooth.characteristic.model_number_string"},
	"2A25": {Name: "Serial Number String", Type: "org.bluetooth.characteristic.serial_number_string"},
	"2A26": {Name: "Firmware Revision String", Type: "org.bluetooth.characteristic.firmware_revision_string"},
	"2A27": {Name: "Hardware Revision String", Type: "org.bluetooth.characteristic.hardware_revision_string"},
	"2A28": {Name: "Software Revision String", Type: "org.bluetooth.characteristic.software_revision_string"},
	"2A29": {Name: "Manufacturer Name String", Type: "org.bluetooth.characteristic.manufacturer_name_string"},
	"2a2A": {Name: "IEEE 11073-20601 Regulatory Certification Data List", Type: "org.bluetooth.characteristic.ieee_11073-20601_regulatory_certification_data_list"},
	"2A2B": {Name: "Current Time", Type: "org.bluetooth.characteristic.current_time"},
	"2A31": {Name: "Scan Refresh", Type: "org.bluetooth.characteristic.scan_refresh"},
	"2A32": {Name: "Boot Keyboard Output Report", Type: "org.bluetooth.characteristic.boot_keyboard_output_report"},
	"2A33": {Name: "Boot Mouse Input Report", Type: "org.bluetooth.characteristic.boot_mouse_input_report"},
	"2A34": {Name: "Glucose Measurement Context", Type: "org.bluetooth.characteristic.glucose_measurement_context"},
	"2A35": {Name: "Blood Pressure Measurement", Type: "org.bluetooth.characteristic.blood_pressure_measurement"},
	"2A36": {Name: "Intermediate Cuff Pressure", Type: "org.bluetooth.characteristic.intermediate_blood_pressure"},
	"2A37": {Name: "Heart Rate Measurement", Type: "org.bluetooth.characteristic.heart_rate_measurement"},
	"2A38": {Name: "Body Sensor Location", Type: "org.bluetooth.characteristic.body_sensor_location"},
	"2A39": {Name: "Heart Rate Control Point", Type: "org.bluetooth.characteristic.heart_rate_control_point"},
	"2A3F": {Name: "Alert Status", Type: "org.bluetooth.characteristic.alert_status"},
	"2A40": {Name: "Ringer Control Point", Type: "org.bluetooth.characteristic.ringer_control_point"},
	"2A41": {Name: "Ringer Setting", Type: "org.bluetooth.characteristic.ringer_setting"},
	"2A42": {Name: "Alert Category ID Bit Mask", Type: "org.bluetooth.characteristic.alert_category_id_bit_mask"},
	"2A43": {Name: "Alert Category ID", Type: "org.bluetooth.characteristic.alert_category_id"},
	"2A44": {Name: "Alert Notification Control Point", Type: "org.bluetooth.characteristic.alert_notification_control_point"},
	"2A45": {Name: "Unread Alert Status", Type: "org.bluetooth.characteristic.unread_alert_status"},
	"2A46": {Name: "New Alert", Type: "org.bluetooth.characteristic.new_alert"},
	"2A47": {Name: "Supported New Alert Category", Type: "org.bluetooth.characteristic.supported_new_alert_category"},
	"2A48": {Name: "Supported Unread Alert Category", Type: "org.bluetooth.characteristic.supported_unread_alert_category"},
	"2A49": {Name: "Blood Pressure Feature", Type: "org.bluetooth.characteristic.blood_pressure_feature"},
	"2A4A": {Name: "HID Information", Type: "org.bluetooth.characteristic.hid_information"},
	"2A4B": {Name: "Report Map", Type: "org.bluetooth.characteristic.report_map"},
	"2A4C": {Name: "HID Control Point", Type: "org.bluetooth.characteristic.hid_control_point"},
	"2A4D": {Name: "Report", Type: "org.bluetooth.characteristic.report"},
	"2A4E": {Name: "Protocol Mode", Type: "org.bluetooth.characteristic.protocol_mode"},
	"2A4F": {Name: "Scan Interval Window", Type: "org.bluetooth.characteristic.scan_interval_window"},
	"2A50": {Name: "PnP ID", Type: "org.bluetooth.characteristic.pnp_id"},
	"2A51": {Name: "Glucose Feature", Type: "org.bluetooth.characteristic.glucose_feature"},
	"2A52": {Name: "Record Access Control Point", Type: "org.bluetooth.characteristic.record_access_control_point"},
	"2A53": {Name: "RSC Measurement", Type: "org.bluetooth.characteristic.rsc_measurement"},
	"2A54": {Name: "RSC Feature", Type: "org.bluetooth.characteristic.rsc_feature"},
	"2A55": {Name: "SC Control Point", Type: "org.bluetooth.characteristic.sc_control_point"},
	"2A5B": {Name: "CSC Measurement", Type: "org.bluetooth.characteristic.csc_measurement"},
	"2A5C": {Name: "CSC Feature", Type: "org.bluetooth.characteristic.csc_feature"},
	"2A5D": {Name: "Sensor Location", Type: "org.bluetooth.characteristic.sensor_location"},
}
