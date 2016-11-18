package main

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type TelecastDeviceType int32

const (
	Telecast_UNKNOWN_DEVICE_TYPE TelecastDeviceType = 0
	Telecast_BGEIGIE_NANO        TelecastDeviceType = 1
	Telecast_SIMPLECAST          TelecastDeviceType = 2
	Telecast_TTAPP               TelecastDeviceType = 3
	Telecast_TTRELAY             TelecastDeviceType = 4
	Telecast_TTGATE              TelecastDeviceType = 5
	Telecast_TTSERVE             TelecastDeviceType = 6
)

var TelecastDeviceType_name = map[int32]string{
	0: "UNKNOWN_DEVICE_TYPE",
	1: "BGEIGIE_NANO",
	2: "SIMPLECAST",
	3: "TTAPP",
	4: "TTRELAY",
	5: "TTGATE",
	6: "TTSERVE",
}
var TelecastDeviceType_value = map[string]int32{
	"UNKNOWN_DEVICE_TYPE": 0,
	"BGEIGIE_NANO":        1,
	"SIMPLECAST":          2,
	"TTAPP":               3,
	"TTRELAY":             4,
	"TTGATE":              5,
	"TTSERVE":             6,
}

func (x TelecastDeviceType) Enum() *TelecastDeviceType {
	p := new(TelecastDeviceType)
	*p = x
	return p
}
func (x TelecastDeviceType) String() string {
	return proto.EnumName(TelecastDeviceType_name, int32(x))
}
func (x *TelecastDeviceType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(TelecastDeviceType_value, data, "TelecastDeviceType")
	if err != nil {
		return err
	}
	*x = TelecastDeviceType(value)
	return nil
}
func (TelecastDeviceType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type TelecastUnit int32

const (
	Telecast_UNKNOWN_UNIT TelecastUnit = 0
	Telecast_CPM          TelecastUnit = 1
)

var TelecastUnit_name = map[int32]string{
	0: "UNKNOWN_UNIT",
	1: "CPM",
}
var TelecastUnit_value = map[string]int32{
	"UNKNOWN_UNIT": 0,
	"CPM":          1,
}

func (x TelecastUnit) Enum() *TelecastUnit {
	p := new(TelecastUnit)
	*p = x
	return p
}
func (x TelecastUnit) String() string {
	return proto.EnumName(TelecastUnit_name, int32(x))
}
func (x *TelecastUnit) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(TelecastUnit_value, data, "TelecastUnit")
	if err != nil {
		return err
	}
	*x = TelecastUnit(value)
	return nil
}
func (TelecastUnit) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 1} }

type Telecast struct {
	DeviceType            *TelecastDeviceType `protobuf:"varint,1,req,name=DeviceType,enum=TelecastDeviceType" json:"DeviceType,omitempty"`
	DeviceIDString        *string             `protobuf:"bytes,2,opt,name=DeviceIDString" json:"DeviceIDString,omitempty"`
	DeviceIDNumber        *uint32             `protobuf:"varint,3,opt,name=DeviceIDNumber" json:"DeviceIDNumber,omitempty"`
	Message               *string             `protobuf:"bytes,4,opt,name=Message" json:"Message,omitempty"`
	CapturedAt            *string             `protobuf:"bytes,5,opt,name=CapturedAt" json:"CapturedAt,omitempty"`
	Unit                  *TelecastUnit       `protobuf:"varint,6,opt,name=Unit,enum=TelecastUnit" json:"Unit,omitempty"`
	Value                 *uint32             `protobuf:"varint,7,opt,name=Value" json:"Value,omitempty"`
	Latitude              *float32            `protobuf:"fixed32,8,opt,name=Latitude" json:"Latitude,omitempty"`
	Longitude             *float32            `protobuf:"fixed32,9,opt,name=Longitude" json:"Longitude,omitempty"`
	Altitude              *uint32             `protobuf:"varint,10,opt,name=Altitude" json:"Altitude,omitempty"`
	BatteryVoltage        *float32            `protobuf:"fixed32,11,opt,name=BatteryVoltage" json:"BatteryVoltage,omitempty"`
	BatterySOC            *float32            `protobuf:"fixed32,12,opt,name=BatterySOC" json:"BatterySOC,omitempty"`
	WirelessSNR           *float32            `protobuf:"fixed32,13,opt,name=WirelessSNR" json:"WirelessSNR,omitempty"`
	EnvTemperature        *float32            `protobuf:"fixed32,14,opt,name=envTemperature" json:"envTemperature,omitempty"`
	EnvHumidity           *float32            `protobuf:"fixed32,15,opt,name=envHumidity" json:"envHumidity,omitempty"`
	RelayDevice1          *uint32             `protobuf:"varint,16,opt,name=RelayDevice1" json:"RelayDevice1,omitempty"`
	RelayDevice2          *uint32             `protobuf:"varint,17,opt,name=RelayDevice2" json:"RelayDevice2,omitempty"`
	RelayDevice3          *uint32             `protobuf:"varint,18,opt,name=RelayDevice3" json:"RelayDevice3,omitempty"`
	RelayDevice4          *uint32             `protobuf:"varint,19,opt,name=RelayDevice4" json:"RelayDevice4,omitempty"`
	RelayDevice5          *uint32             `protobuf:"varint,20,opt,name=RelayDevice5" json:"RelayDevice5,omitempty"`
	Cpm0                  *uint32             `protobuf:"varint,21,opt,name=cpm0" json:"cpm0,omitempty"`
	Cpm1                  *uint32             `protobuf:"varint,22,opt,name=cpm1" json:"cpm1,omitempty"`
	StatsUptimeMinutes    *uint32             `protobuf:"varint,23,opt,name=stats_uptime_minutes" json:"stats_uptime_minutes,omitempty"`
	StatsAppVersion       *string             `protobuf:"bytes,24,opt,name=stats_app_version" json:"stats_app_version,omitempty"`
	StatsDeviceParams     *string             `protobuf:"bytes,25,opt,name=stats_device_params" json:"stats_device_params,omitempty"`
	StatsTransmittedBytes *uint32             `protobuf:"varint,26,opt,name=stats_transmitted_bytes" json:"stats_transmitted_bytes,omitempty"`
	StatsReceivedBytes    *uint32             `protobuf:"varint,27,opt,name=stats_received_bytes" json:"stats_received_bytes,omitempty"`
	StatsOneshots         *uint32             `protobuf:"varint,28,opt,name=stats_oneshots" json:"stats_oneshots,omitempty"`
	StatsCommsResets      *uint32             `protobuf:"varint,29,opt,name=stats_comms_resets" json:"stats_comms_resets,omitempty"`
	PmsPm01_0             *uint32             `protobuf:"varint,30,opt,name=pms_pm01_0" json:"pms_pm01_0,omitempty"`
	PmsPm02_5             *uint32             `protobuf:"varint,31,opt,name=pms_pm02_5" json:"pms_pm02_5,omitempty"`
	PmsPm10_0             *uint32             `protobuf:"varint,32,opt,name=pms_pm10_0" json:"pms_pm10_0,omitempty"`
	PmsC00_30             *uint32             `protobuf:"varint,33,opt,name=pms_c00_30" json:"pms_c00_30,omitempty"`
	PmsC00_50             *uint32             `protobuf:"varint,34,opt,name=pms_c00_50" json:"pms_c00_50,omitempty"`
	PmsC01_00             *uint32             `protobuf:"varint,35,opt,name=pms_c01_00" json:"pms_c01_00,omitempty"`
	PmsC02_50             *uint32             `protobuf:"varint,36,opt,name=pms_c02_50" json:"pms_c02_50,omitempty"`
	PmsC05_00             *uint32             `protobuf:"varint,37,opt,name=pms_c05_00" json:"pms_c05_00,omitempty"`
	PmsC10_00             *uint32             `protobuf:"varint,38,opt,name=pms_c10_00" json:"pms_c10_00,omitempty"`
	PmsCsecs              *uint32             `protobuf:"varint,39,opt,name=pms_csecs" json:"pms_csecs,omitempty"`
	OpcPm01_0             *float32            `protobuf:"fixed32,40,opt,name=opc_pm01_0" json:"opc_pm01_0,omitempty"`
	OpcPm02_5             *float32            `protobuf:"fixed32,41,opt,name=opc_pm02_5" json:"opc_pm02_5,omitempty"`
	OpcPm10_0             *float32            `protobuf:"fixed32,42,opt,name=opc_pm10_0" json:"opc_pm10_0,omitempty"`
	OpcC00_38             *uint32             `protobuf:"varint,43,opt,name=opc_c00_38" json:"opc_c00_38,omitempty"`
	OpcC00_54             *uint32             `protobuf:"varint,44,opt,name=opc_c00_54" json:"opc_c00_54,omitempty"`
	OpcC01_00             *uint32             `protobuf:"varint,45,opt,name=opc_c01_00" json:"opc_c01_00,omitempty"`
	OpcC02_10             *uint32             `protobuf:"varint,46,opt,name=opc_c02_10" json:"opc_c02_10,omitempty"`
	OpcC05_00             *uint32             `protobuf:"varint,47,opt,name=opc_c05_00" json:"opc_c05_00,omitempty"`
	OpcC10_00             *uint32             `protobuf:"varint,48,opt,name=opc_c10_00" json:"opc_c10_00,omitempty"`
	OpcCsecs              *uint32             `protobuf:"varint,49,opt,name=opc_csecs" json:"opc_csecs,omitempty"`
	EnvPressure           *float32            `protobuf:"fixed32,50,opt,name=envPressure" json:"envPressure,omitempty"`
	StatsCommsPowerFails  *uint32             `protobuf:"varint,51,opt,name=stats_comms_power_fails" json:"stats_comms_power_fails,omitempty"`
	BatteryCurrent        *float32            `protobuf:"fixed32,52,opt,name=BatteryCurrent" json:"BatteryCurrent,omitempty"`
	XXX_unrecognized      []byte              `json:"-"`
}

func (m *Telecast) Reset()                    { *m = Telecast{} }
func (m *Telecast) String() string            { return proto.CompactTextString(m) }
func (*Telecast) ProtoMessage()               {}
func (*Telecast) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Telecast) GetDeviceType() TelecastDeviceType {
	if m != nil && m.DeviceType != nil {
		return *m.DeviceType
	}
	return Telecast_UNKNOWN_DEVICE_TYPE
}

func (m *Telecast) GetDeviceIDString() string {
	if m != nil && m.DeviceIDString != nil {
		return *m.DeviceIDString
	}
	return ""
}

func (m *Telecast) GetDeviceIDNumber() uint32 {
	if m != nil && m.DeviceIDNumber != nil {
		return *m.DeviceIDNumber
	}
	return 0
}

func (m *Telecast) GetMessage() string {
	if m != nil && m.Message != nil {
		return *m.Message
	}
	return ""
}

func (m *Telecast) GetCapturedAt() string {
	if m != nil && m.CapturedAt != nil {
		return *m.CapturedAt
	}
	return ""
}

func (m *Telecast) GetUnit() TelecastUnit {
	if m != nil && m.Unit != nil {
		return *m.Unit
	}
	return Telecast_UNKNOWN_UNIT
}

func (m *Telecast) GetValue() uint32 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

func (m *Telecast) GetLatitude() float32 {
	if m != nil && m.Latitude != nil {
		return *m.Latitude
	}
	return 0
}

func (m *Telecast) GetLongitude() float32 {
	if m != nil && m.Longitude != nil {
		return *m.Longitude
	}
	return 0
}

func (m *Telecast) GetAltitude() uint32 {
	if m != nil && m.Altitude != nil {
		return *m.Altitude
	}
	return 0
}

func (m *Telecast) GetBatteryVoltage() float32 {
	if m != nil && m.BatteryVoltage != nil {
		return *m.BatteryVoltage
	}
	return 0
}

func (m *Telecast) GetBatterySOC() float32 {
	if m != nil && m.BatterySOC != nil {
		return *m.BatterySOC
	}
	return 0
}

func (m *Telecast) GetWirelessSNR() float32 {
	if m != nil && m.WirelessSNR != nil {
		return *m.WirelessSNR
	}
	return 0
}

func (m *Telecast) GetEnvTemperature() float32 {
	if m != nil && m.EnvTemperature != nil {
		return *m.EnvTemperature
	}
	return 0
}

func (m *Telecast) GetEnvHumidity() float32 {
	if m != nil && m.EnvHumidity != nil {
		return *m.EnvHumidity
	}
	return 0
}

func (m *Telecast) GetRelayDevice1() uint32 {
	if m != nil && m.RelayDevice1 != nil {
		return *m.RelayDevice1
	}
	return 0
}

func (m *Telecast) GetRelayDevice2() uint32 {
	if m != nil && m.RelayDevice2 != nil {
		return *m.RelayDevice2
	}
	return 0
}

func (m *Telecast) GetRelayDevice3() uint32 {
	if m != nil && m.RelayDevice3 != nil {
		return *m.RelayDevice3
	}
	return 0
}

func (m *Telecast) GetRelayDevice4() uint32 {
	if m != nil && m.RelayDevice4 != nil {
		return *m.RelayDevice4
	}
	return 0
}

func (m *Telecast) GetRelayDevice5() uint32 {
	if m != nil && m.RelayDevice5 != nil {
		return *m.RelayDevice5
	}
	return 0
}

func (m *Telecast) GetCpm0() uint32 {
	if m != nil && m.Cpm0 != nil {
		return *m.Cpm0
	}
	return 0
}

func (m *Telecast) GetCpm1() uint32 {
	if m != nil && m.Cpm1 != nil {
		return *m.Cpm1
	}
	return 0
}

func (m *Telecast) GetStatsUptimeMinutes() uint32 {
	if m != nil && m.StatsUptimeMinutes != nil {
		return *m.StatsUptimeMinutes
	}
	return 0
}

func (m *Telecast) GetStatsAppVersion() string {
	if m != nil && m.StatsAppVersion != nil {
		return *m.StatsAppVersion
	}
	return ""
}

func (m *Telecast) GetStatsDeviceParams() string {
	if m != nil && m.StatsDeviceParams != nil {
		return *m.StatsDeviceParams
	}
	return ""
}

func (m *Telecast) GetStatsTransmittedBytes() uint32 {
	if m != nil && m.StatsTransmittedBytes != nil {
		return *m.StatsTransmittedBytes
	}
	return 0
}

func (m *Telecast) GetStatsReceivedBytes() uint32 {
	if m != nil && m.StatsReceivedBytes != nil {
		return *m.StatsReceivedBytes
	}
	return 0
}

func (m *Telecast) GetStatsOneshots() uint32 {
	if m != nil && m.StatsOneshots != nil {
		return *m.StatsOneshots
	}
	return 0
}

func (m *Telecast) GetStatsCommsResets() uint32 {
	if m != nil && m.StatsCommsResets != nil {
		return *m.StatsCommsResets
	}
	return 0
}

func (m *Telecast) GetPmsPm01_0() uint32 {
	if m != nil && m.PmsPm01_0 != nil {
		return *m.PmsPm01_0
	}
	return 0
}

func (m *Telecast) GetPmsPm02_5() uint32 {
	if m != nil && m.PmsPm02_5 != nil {
		return *m.PmsPm02_5
	}
	return 0
}

func (m *Telecast) GetPmsPm10_0() uint32 {
	if m != nil && m.PmsPm10_0 != nil {
		return *m.PmsPm10_0
	}
	return 0
}

func (m *Telecast) GetPmsC00_30() uint32 {
	if m != nil && m.PmsC00_30 != nil {
		return *m.PmsC00_30
	}
	return 0
}

func (m *Telecast) GetPmsC00_50() uint32 {
	if m != nil && m.PmsC00_50 != nil {
		return *m.PmsC00_50
	}
	return 0
}

func (m *Telecast) GetPmsC01_00() uint32 {
	if m != nil && m.PmsC01_00 != nil {
		return *m.PmsC01_00
	}
	return 0
}

func (m *Telecast) GetPmsC02_50() uint32 {
	if m != nil && m.PmsC02_50 != nil {
		return *m.PmsC02_50
	}
	return 0
}

func (m *Telecast) GetPmsC05_00() uint32 {
	if m != nil && m.PmsC05_00 != nil {
		return *m.PmsC05_00
	}
	return 0
}

func (m *Telecast) GetPmsC10_00() uint32 {
	if m != nil && m.PmsC10_00 != nil {
		return *m.PmsC10_00
	}
	return 0
}

func (m *Telecast) GetPmsCsecs() uint32 {
	if m != nil && m.PmsCsecs != nil {
		return *m.PmsCsecs
	}
	return 0
}

func (m *Telecast) GetOpcPm01_0() float32 {
	if m != nil && m.OpcPm01_0 != nil {
		return *m.OpcPm01_0
	}
	return 0
}

func (m *Telecast) GetOpcPm02_5() float32 {
	if m != nil && m.OpcPm02_5 != nil {
		return *m.OpcPm02_5
	}
	return 0
}

func (m *Telecast) GetOpcPm10_0() float32 {
	if m != nil && m.OpcPm10_0 != nil {
		return *m.OpcPm10_0
	}
	return 0
}

func (m *Telecast) GetOpcC00_38() uint32 {
	if m != nil && m.OpcC00_38 != nil {
		return *m.OpcC00_38
	}
	return 0
}

func (m *Telecast) GetOpcC00_54() uint32 {
	if m != nil && m.OpcC00_54 != nil {
		return *m.OpcC00_54
	}
	return 0
}

func (m *Telecast) GetOpcC01_00() uint32 {
	if m != nil && m.OpcC01_00 != nil {
		return *m.OpcC01_00
	}
	return 0
}

func (m *Telecast) GetOpcC02_10() uint32 {
	if m != nil && m.OpcC02_10 != nil {
		return *m.OpcC02_10
	}
	return 0
}

func (m *Telecast) GetOpcC05_00() uint32 {
	if m != nil && m.OpcC05_00 != nil {
		return *m.OpcC05_00
	}
	return 0
}

func (m *Telecast) GetOpcC10_00() uint32 {
	if m != nil && m.OpcC10_00 != nil {
		return *m.OpcC10_00
	}
	return 0
}

func (m *Telecast) GetOpcCsecs() uint32 {
	if m != nil && m.OpcCsecs != nil {
		return *m.OpcCsecs
	}
	return 0
}

func (m *Telecast) GetEnvPressure() float32 {
	if m != nil && m.EnvPressure != nil {
		return *m.EnvPressure
	}
	return 0
}

func (m *Telecast) GetStatsCommsPowerFails() uint32 {
	if m != nil && m.StatsCommsPowerFails != nil {
		return *m.StatsCommsPowerFails
	}
	return 0
}

func (m *Telecast) GetBatteryCurrent() float32 {
	if m != nil && m.BatteryCurrent != nil {
		return *m.BatteryCurrent
	}
	return 0
}

func init() {
	proto.RegisterType((*Telecast)(nil), "Telecast")
	proto.RegisterEnum("TelecastDeviceType", TelecastDeviceType_name, TelecastDeviceType_value)
	proto.RegisterEnum("TelecastUnit", TelecastUnit_name, TelecastUnit_value)
}

var fileDescriptor0 = []byte{
	// 717 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x6c, 0x94, 0xed, 0x53, 0xda, 0x4a,
	0x14, 0xc6, 0x2f, 0x08, 0x02, 0x47, 0xc4, 0x18, 0xbc, 0x72, 0x7c, 0xbb, 0x2a, 0xb7, 0xb6, 0xf6,
	0x8d, 0x06, 0x94, 0x99, 0x7e, 0x45, 0xcc, 0x58, 0xa6, 0x18, 0x19, 0x88, 0x38, 0x7e, 0xca, 0xc4,
	0xb0, 0xb5, 0x99, 0x21, 0x2f, 0x93, 0xdd, 0xd0, 0xf2, 0xa7, 0xf5, 0xbf, 0xeb, 0x66, 0x63, 0x18,
	0xe2, 0xf0, 0x71, 0x7f, 0xcf, 0xb3, 0xbb, 0xe7, 0xc9, 0x39, 0x1b, 0xa8, 0x30, 0x32, 0x25, 0x6c,
	0xee, 0x93, 0x86, 0x1f, 0x78, 0xcc, 0x93, 0x8b, 0xc9, 0xba, 0xfe, 0x07, 0xa0, 0xa8, 0xf3, 0x85,
	0x65, 0x52, 0x26, 0x37, 0x01, 0xae, 0xc9, 0xcc, 0xb6, 0x88, 0xce, 0x25, 0xcc, 0x9c, 0x64, 0xcf,
	0x2b, 0xad, 0xa3, 0xc6, 0x62, 0x6f, 0xe2, 0x6b, 0x4c, 0x16, 0x26, 0x79, 0x17, 0x2a, 0xf1, 0x96,
	0xde, 0xf5, 0x88, 0x05, 0xb6, 0xfb, 0x8c, 0xd9, 0x93, 0xcc, 0x79, 0x69, 0x99, 0x6b, 0xa1, 0xf3,
	0x44, 0x02, 0x5c, 0xe3, 0x7c, 0x53, 0xde, 0x82, 0xc2, 0x2d, 0xa1, 0xd4, 0x7c, 0x26, 0x98, 0x13,
	0x46, 0x19, 0xa0, 0x6b, 0xfa, 0x2c, 0x0c, 0xc8, 0xa4, 0xc3, 0x30, 0x2f, 0xd8, 0x19, 0xe4, 0xee,
	0x5d, 0x9b, 0xe1, 0x3a, 0x5f, 0x55, 0x5a, 0xb5, 0x15, 0x15, 0x84, 0x5c, 0x96, 0x37, 0x21, 0x3f,
	0x36, 0xa7, 0x21, 0xc1, 0x82, 0x38, 0x5a, 0x82, 0x62, 0xdf, 0x64, 0x36, 0x0b, 0x27, 0x04, 0x8b,
	0x9c, 0x64, 0xe5, 0x6d, 0x28, 0xf5, 0x3d, 0xf7, 0x39, 0x46, 0x25, 0x81, 0xb8, 0xa9, 0x33, 0x7d,
	0x31, 0x81, 0xd8, 0xc6, 0x2b, 0xbd, 0x32, 0x19, 0x23, 0xc1, 0x7c, 0xec, 0x4d, 0x59, 0x54, 0xd8,
	0x86, 0x70, 0xf2, 0xc2, 0x5e, 0xf8, 0xe8, 0xae, 0x8b, 0x65, 0xc1, 0xaa, 0xb0, 0xf1, 0x60, 0x07,
	0xbc, 0x08, 0x4a, 0x47, 0xda, 0x10, 0x37, 0x05, 0xe4, 0x07, 0x10, 0x77, 0xa6, 0x13, 0xc7, 0x27,
	0x81, 0x19, 0x05, 0xc1, 0x4a, 0x62, 0xe6, 0xfc, 0x5b, 0xe8, 0xd8, 0x13, 0x9b, 0xcd, 0x71, 0x4b,
	0xc0, 0x1d, 0x28, 0x0f, 0xc9, 0xd4, 0x9c, 0xc7, 0x1f, 0xa7, 0x89, 0x92, 0xa8, 0x21, 0x4d, 0x5b,
	0xb8, 0xbd, 0x82, 0x5e, 0xa0, 0xbc, 0x82, 0x5e, 0x62, 0x75, 0x05, 0x6d, 0xe3, 0x8e, 0xa0, 0x65,
	0xc8, 0x59, 0xbe, 0xa3, 0xe0, 0xbf, 0x4b, 0xab, 0x26, 0xee, 0x8a, 0xd5, 0x21, 0xec, 0x50, 0x66,
	0x32, 0x6a, 0x84, 0x3e, 0xb3, 0x1d, 0x62, 0x38, 0xb6, 0x1b, 0x32, 0x42, 0xb1, 0x26, 0xd4, 0x3d,
	0xd8, 0x8e, 0x55, 0xd3, 0xf7, 0x8d, 0x19, 0x09, 0xa8, 0xed, 0xb9, 0x88, 0xa2, 0x3b, 0x07, 0x50,
	0x8d, 0xa5, 0x78, 0x0c, 0x0c, 0xdf, 0x0c, 0x4c, 0x87, 0xe2, 0x9e, 0x10, 0x8f, 0xa1, 0x16, 0x8b,
	0x2c, 0x30, 0x5d, 0xea, 0xd8, 0xfc, 0x03, 0x4e, 0x8c, 0xa7, 0x79, 0x74, 0xf0, 0x7e, 0xfa, 0xda,
	0x80, 0x58, 0xc4, 0x9e, 0x2d, 0xd4, 0x83, 0xa4, 0x19, 0xb1, 0xea, 0xb9, 0x84, 0xfe, 0xf4, 0x18,
	0xc5, 0x43, 0xc1, 0xf7, 0x41, 0x8e, 0xb9, 0xe5, 0x39, 0x4e, 0xb4, 0x97, 0x12, 0xae, 0x1d, 0x09,
	0x8d, 0x37, 0xca, 0xe7, 0x8c, 0xe7, 0x6c, 0x1a, 0x0a, 0xfe, 0xf7, 0x8a, 0xb5, 0x8c, 0x36, 0x1e,
	0xa7, 0x59, 0x53, 0xe1, 0xbe, 0x93, 0x65, 0x66, 0x29, 0x8a, 0x71, 0xa1, 0xe0, 0xe9, 0x6b, 0xd6,
	0x56, 0xb0, 0x9e, 0x66, 0xfc, 0x0a, 0x05, 0xff, 0x4f, 0xb3, 0x56, 0xe4, 0x7b, 0x93, 0x66, 0xed,
	0xc8, 0x77, 0x96, 0x62, 0xd1, 0xb5, 0x0a, 0xbe, 0x15, 0x8c, 0x4f, 0xa6, 0x60, 0x94, 0x58, 0x14,
	0xdf, 0x25, 0x36, 0xcf, 0xb7, 0x92, 0x18, 0xe7, 0xc9, 0x0c, 0xbe, 0xb0, 0x28, 0xc6, 0xfb, 0x34,
	0x13, 0x31, 0x3e, 0x2c, 0x33, 0x11, 0xe3, 0x2b, 0x7e, 0x5c, 0x3e, 0x4f, 0xc4, 0xb8, 0xc4, 0x4f,
	0x69, 0x26, 0x62, 0x7c, 0x4e, 0xb3, 0x96, 0xd1, 0x54, 0xb0, 0x91, 0x66, 0x22, 0xc6, 0x97, 0x14,
	0x8b, 0x63, 0x28, 0x49, 0x0c, 0xc1, 0x44, 0x8c, 0xa6, 0x40, 0xf1, 0xd4, 0x0f, 0x78, 0x87, 0x68,
	0xf4, 0x14, 0x5a, 0xa2, 0xbe, 0xc5, 0x54, 0xc4, 0xed, 0xf3, 0xbd, 0x5f, 0x24, 0x30, 0x7e, 0x98,
	0xf6, 0x94, 0xe2, 0xc5, 0xab, 0x47, 0xd8, 0x0d, 0x83, 0x80, 0xb8, 0x0c, 0x2f, 0xa3, 0x8d, 0xf5,
	0xdf, 0x00, 0x4b, 0x3f, 0x9b, 0x1a, 0x54, 0xef, 0xb5, 0xef, 0xda, 0xdd, 0x83, 0x66, 0x5c, 0xab,
	0xe3, 0x5e, 0x57, 0x35, 0xf4, 0xc7, 0x81, 0x2a, 0xfd, 0xc3, 0x5f, 0x75, 0xf9, 0xea, 0x46, 0xed,
	0xdd, 0xf4, 0x54, 0x43, 0xeb, 0x68, 0x77, 0x52, 0x46, 0xae, 0x00, 0x8c, 0x7a, 0xb7, 0x83, 0xbe,
	0xda, 0xed, 0x8c, 0x74, 0x29, 0x2b, 0x97, 0x20, 0xaf, 0xeb, 0x9d, 0xc1, 0x40, 0x5a, 0x93, 0x37,
	0xa0, 0xa0, 0xeb, 0x43, 0xb5, 0xdf, 0x79, 0x94, 0x72, 0x3c, 0xd4, 0xba, 0xae, 0xdf, 0x74, 0x74,
	0x55, 0xca, 0xc7, 0xc2, 0x48, 0x1d, 0x8e, 0x55, 0x69, 0xbd, 0x7e, 0x0a, 0x39, 0xf1, 0x93, 0xe1,
	0x47, 0x27, 0x77, 0xde, 0x6b, 0x3d, 0x9d, 0x5f, 0x56, 0x80, 0xb5, 0xee, 0xe0, 0x56, 0xca, 0xfc,
	0x0d, 0x00, 0x00, 0xff, 0xff, 0x50, 0x74, 0x1a, 0x43, 0x56, 0x05, 0x00, 0x00,
}

func main() {}
func test(msg *Telecast) {}
