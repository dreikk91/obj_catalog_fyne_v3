export namespace main {
	
	export class OperatorDBSettings {
	    FirebirdEnabled: boolean;
	    FirebirdUser: string;
	    FirebirdPassword: string;
	    FirebirdHost: string;
	    FirebirdPort: string;
	    FirebirdPath: string;
	    FirebirdParams: string;
	    PhoenixEnabled: boolean;
	    PhoenixUser: string;
	    PhoenixPassword: string;
	    PhoenixHost: string;
	    PhoenixPort: string;
	    PhoenixInstance: string;
	    PhoenixDatabase: string;
	    PhoenixParams: string;
	    CASLEnabled: boolean;
	    CASLBaseURL: string;
	    CASLToken: string;
	    CASLEmail: string;
	    CASLPass: string;
	    CASLPultID: number;
	    Mode: string;
	
	    static createFrom(source: any = {}) {
	        return new OperatorDBSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FirebirdEnabled = source["FirebirdEnabled"];
	        this.FirebirdUser = source["FirebirdUser"];
	        this.FirebirdPassword = source["FirebirdPassword"];
	        this.FirebirdHost = source["FirebirdHost"];
	        this.FirebirdPort = source["FirebirdPort"];
	        this.FirebirdPath = source["FirebirdPath"];
	        this.FirebirdParams = source["FirebirdParams"];
	        this.PhoenixEnabled = source["PhoenixEnabled"];
	        this.PhoenixUser = source["PhoenixUser"];
	        this.PhoenixPassword = source["PhoenixPassword"];
	        this.PhoenixHost = source["PhoenixHost"];
	        this.PhoenixPort = source["PhoenixPort"];
	        this.PhoenixInstance = source["PhoenixInstance"];
	        this.PhoenixDatabase = source["PhoenixDatabase"];
	        this.PhoenixParams = source["PhoenixParams"];
	        this.CASLEnabled = source["CASLEnabled"];
	        this.CASLBaseURL = source["CASLBaseURL"];
	        this.CASLToken = source["CASLToken"];
	        this.CASLEmail = source["CASLEmail"];
	        this.CASLPass = source["CASLPass"];
	        this.CASLPultID = source["CASLPultID"];
	        this.Mode = source["Mode"];
	    }
	}

}

export namespace v1 {
	
	export class AlarmItem {
	    ID: number;
	    Source: string;
	    ObjectID: number;
	    ObjectNativeID: string;
	    ObjectNumber: string;
	    ObjectName: string;
	    Address: string;
	    Time: string;
	    Details: string;
	    TypeCode: string;
	    TypeText: string;
	    ZoneNumber: number;
	    ZoneName: string;
	    IsProcessed: boolean;
	    ProcessedBy: string;
	    ProcessNote: string;
	    IsInProgress: boolean;
	    InProgressBy: string;
	    IsOwnedByMe: boolean;
	    CanTakeOver: boolean;
	    CanProcess: boolean;
	    VisualSeverity: string;
	
	    static createFrom(source: any = {}) {
	        return new AlarmItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Source = source["Source"];
	        this.ObjectID = source["ObjectID"];
	        this.ObjectNativeID = source["ObjectNativeID"];
	        this.ObjectNumber = source["ObjectNumber"];
	        this.ObjectName = source["ObjectName"];
	        this.Address = source["Address"];
	        this.Time = source["Time"];
	        this.Details = source["Details"];
	        this.TypeCode = source["TypeCode"];
	        this.TypeText = source["TypeText"];
	        this.ZoneNumber = source["ZoneNumber"];
	        this.ZoneName = source["ZoneName"];
	        this.IsProcessed = source["IsProcessed"];
	        this.ProcessedBy = source["ProcessedBy"];
	        this.ProcessNote = source["ProcessNote"];
	        this.IsInProgress = source["IsInProgress"];
	        this.InProgressBy = source["InProgressBy"];
	        this.IsOwnedByMe = source["IsOwnedByMe"];
	        this.CanTakeOver = source["CanTakeOver"];
	        this.CanProcess = source["CanProcess"];
	        this.VisualSeverity = source["VisualSeverity"];
	    }
	}
	export class AlarmGroup {
	    GroupID: string;
	    Source: string;
	    ObjectID: number;
	    ObjectNativeID: string;
	    ObjectNumber: string;
	    ObjectName: string;
	    Address: string;
	    AlertLevel: number;
	    LatestTime: string;
	    Primary: AlarmItem;
	    Items: AlarmItem[];
	
	    static createFrom(source: any = {}) {
	        return new AlarmGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.GroupID = source["GroupID"];
	        this.Source = source["Source"];
	        this.ObjectID = source["ObjectID"];
	        this.ObjectNativeID = source["ObjectNativeID"];
	        this.ObjectNumber = source["ObjectNumber"];
	        this.ObjectName = source["ObjectName"];
	        this.Address = source["Address"];
	        this.AlertLevel = source["AlertLevel"];
	        this.LatestTime = source["LatestTime"];
	        this.Primary = this.convertValues(source["Primary"], AlarmItem);
	        this.Items = this.convertValues(source["Items"], AlarmItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AlarmGroupActionRequest {
	    GroupID: string;
	
	    static createFrom(source: any = {}) {
	        return new AlarmGroupActionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.GroupID = source["GroupID"];
	    }
	}
	
	export class AlarmPickRequest {
	    User: string;
	
	    static createFrom(source: any = {}) {
	        return new AlarmPickRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.User = source["User"];
	    }
	}
	export class AlarmProcessRequest {
	    User: string;
	    CauseCode: string;
	    Note: string;
	
	    static createFrom(source: any = {}) {
	        return new AlarmProcessRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.User = source["User"];
	        this.CauseCode = source["CauseCode"];
	        this.Note = source["Note"];
	    }
	}
	export class AlarmProcessingOption {
	    Code: string;
	    Label: string;
	
	    static createFrom(source: any = {}) {
	        return new AlarmProcessingOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Code = source["Code"];
	        this.Label = source["Label"];
	    }
	}
	export class SourceCapability {
	    Source: string;
	    DisplayName: string;
	    ReadObjects: boolean;
	    ReadObjectDetails: boolean;
	    ReadEvents: boolean;
	    ReadAlarms: boolean;
	    CreateObject: boolean;
	    UpdateObject: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SourceCapability(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Source = source["Source"];
	        this.DisplayName = source["DisplayName"];
	        this.ReadObjects = source["ReadObjects"];
	        this.ReadObjectDetails = source["ReadObjectDetails"];
	        this.ReadEvents = source["ReadEvents"];
	        this.ReadAlarms = source["ReadAlarms"];
	        this.CreateObject = source["CreateObject"];
	        this.UpdateObject = source["UpdateObject"];
	    }
	}
	export class Capabilities {
	    Sources: SourceCapability[];
	
	    static createFrom(source: any = {}) {
	        return new Capabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Sources = this.convertValues(source["Sources"], SourceCapability);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Contact {
	    Name: string;
	    Position: string;
	    Phone: string;
	    Priority: number;
	    CodeWord: string;
	    GroupID: string;
	    GroupNumber: number;
	    GroupName: string;
	    GroupStateText: string;
	
	    static createFrom(source: any = {}) {
	        return new Contact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Position = source["Position"];
	        this.Phone = source["Phone"];
	        this.Priority = source["Priority"];
	        this.CodeWord = source["CodeWord"];
	        this.GroupID = source["GroupID"];
	        this.GroupNumber = source["GroupNumber"];
	        this.GroupName = source["GroupName"];
	        this.GroupStateText = source["GroupStateText"];
	    }
	}
	export class EventItem {
	    ID: number;
	    Source: string;
	    ObjectID: number;
	    ObjectNativeID: string;
	    ObjectNumber: string;
	    ObjectName: string;
	    Time: string;
	    TypeCode: string;
	    TypeText: string;
	    ZoneNumber: number;
	    Details: string;
	    UserName: string;
	    VisualSeverity: string;
	
	    static createFrom(source: any = {}) {
	        return new EventItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Source = source["Source"];
	        this.ObjectID = source["ObjectID"];
	        this.ObjectNativeID = source["ObjectNativeID"];
	        this.ObjectNumber = source["ObjectNumber"];
	        this.ObjectName = source["ObjectName"];
	        this.Time = source["Time"];
	        this.TypeCode = source["TypeCode"];
	        this.TypeText = source["TypeText"];
	        this.ZoneNumber = source["ZoneNumber"];
	        this.Details = source["Details"];
	        this.UserName = source["UserName"];
	        this.VisualSeverity = source["VisualSeverity"];
	    }
	}
	export class EventPageResponse {
	    items: EventItem[];
	    totalCount: number;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EventPageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], EventItem);
	        this.totalCount = source["totalCount"];
	        this.hasMore = source["hasMore"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Zone {
	    Number: number;
	    Name: string;
	    SensorType: string;
	    Status: string;
	    GroupID: string;
	    GroupNumber: number;
	    GroupName: string;
	    GroupStateText: string;
	
	    static createFrom(source: any = {}) {
	        return new Zone(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Number = source["Number"];
	        this.Name = source["Name"];
	        this.SensorType = source["SensorType"];
	        this.Status = source["Status"];
	        this.GroupID = source["GroupID"];
	        this.GroupNumber = source["GroupNumber"];
	        this.GroupName = source["GroupName"];
	        this.GroupStateText = source["GroupStateText"];
	    }
	}
	export class ObjectSummary {
	    ID: number;
	    Source: string;
	    NativeID: string;
	    DisplayNumber: string;
	    Name: string;
	    Address: string;
	    ContractNumber: string;
	    Phone: string;
	    StatusCode: string;
	    StatusText: string;
	    DeviceType: string;
	    PanelMark: string;
	    SignalStrength: string;
	    SIM1: string;
	    SIM2: string;
	    LastTestTime: string;
	    LastMessageTime: string;
	    GuardStatus: string;
	    ConnectionStatus: string;
	    MonitoringStatus: string;
	    HasAssignment: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ObjectSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Source = source["Source"];
	        this.NativeID = source["NativeID"];
	        this.DisplayNumber = source["DisplayNumber"];
	        this.Name = source["Name"];
	        this.Address = source["Address"];
	        this.ContractNumber = source["ContractNumber"];
	        this.Phone = source["Phone"];
	        this.StatusCode = source["StatusCode"];
	        this.StatusText = source["StatusText"];
	        this.DeviceType = source["DeviceType"];
	        this.PanelMark = source["PanelMark"];
	        this.SignalStrength = source["SignalStrength"];
	        this.SIM1 = source["SIM1"];
	        this.SIM2 = source["SIM2"];
	        this.LastTestTime = source["LastTestTime"];
	        this.LastMessageTime = source["LastMessageTime"];
	        this.GuardStatus = source["GuardStatus"];
	        this.ConnectionStatus = source["ConnectionStatus"];
	        this.MonitoringStatus = source["MonitoringStatus"];
	        this.HasAssignment = source["HasAssignment"];
	    }
	}
	export class ObjectDetails {
	    Summary: ObjectSummary;
	    GSMLevel: number;
	    PowerSource: string;
	    AutoTestHours: number;
	    SubServerA: string;
	    SubServerB: string;
	    ChannelCode: number;
	    AKBState: number;
	    PowerFault: number;
	    TestControl: boolean;
	    TestIntervalMin: number;
	    Phones: string;
	    Notes: string;
	    Location: string;
	    LaunchDate: string;
	    ExternalSignal: string;
	    ExternalTestMessage: string;
	    ExternalLastTest: string;
	    ExternalLastMessage: string;
	    Zones: Zone[];
	    Contacts: Contact[];
	    Events: EventItem[];
	
	    static createFrom(source: any = {}) {
	        return new ObjectDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Summary = this.convertValues(source["Summary"], ObjectSummary);
	        this.GSMLevel = source["GSMLevel"];
	        this.PowerSource = source["PowerSource"];
	        this.AutoTestHours = source["AutoTestHours"];
	        this.SubServerA = source["SubServerA"];
	        this.SubServerB = source["SubServerB"];
	        this.ChannelCode = source["ChannelCode"];
	        this.AKBState = source["AKBState"];
	        this.PowerFault = source["PowerFault"];
	        this.TestControl = source["TestControl"];
	        this.TestIntervalMin = source["TestIntervalMin"];
	        this.Phones = source["Phones"];
	        this.Notes = source["Notes"];
	        this.Location = source["Location"];
	        this.LaunchDate = source["LaunchDate"];
	        this.ExternalSignal = source["ExternalSignal"];
	        this.ExternalTestMessage = source["ExternalTestMessage"];
	        this.ExternalLastTest = source["ExternalLastTest"];
	        this.ExternalLastMessage = source["ExternalLastMessage"];
	        this.Zones = this.convertValues(source["Zones"], Zone);
	        this.Contacts = this.convertValues(source["Contacts"], Contact);
	        this.Events = this.convertValues(source["Events"], EventItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ResponseGroup {
	    ID: string;
	    Name: string;
	    Callsign: string;
	    Phone: string;
	
	    static createFrom(source: any = {}) {
	        return new ResponseGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Callsign = source["Callsign"];
	        this.Phone = source["Phone"];
	    }
	}
	

}

