package main

import (
	"fmt"
	"testing"
)

var data = `<?xml version="1.0" encoding="UTF-8"?>
    <TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject> 
			<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/> 
			<TrkAttr name="PRODUCTNAME" val="CFT"/> 
			<TrkAttr name="PRODUCTIPADDR" val="cft"/> 
		</TrkObject> 
	</TrkDescriptor>`

var dataISO8859 = `<?xml version="1.0" encoding="ISO-8859-1"?>
	<TrkDescriptor>
		<TrkXML version="1.0"/>
		<TrkObject>
			<TrkIdentifier type="CycleLink"/>
			<TrkAttr name="EVENTDATE" val="2019.09.03"/>
			<TrkAttr name="EVENTTIME" val="15:20:23"/>
			<TrkAttr name="PARENTOBJECT" val="XFBTransfer"/>
			<TrkAttr name="PARENTCYCLEID" val="XFhrW9CShrOM8gX9S2AmK7+1dY0qMX"/>
			<TrkAttr name="CHILDOBJECT" val="XFBTransfer"/>
			<TrkAttr name="CHILDCYCLEID" val="XFS34me0RWqp1cBQb/lsfLvlKpamMX"/>
			</TrkObject>
	</TrkDescriptor>`

var dataInvalidXML = `<?xml version="1.0" encoding="UTF-8"?>
	<TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject> 
			<TrkIdentifier TYPE="Event" NAME="XFBLog" VERSION="1.0"/> 
			<TrkAttr name="PRODUCTNAME" val="CFT"/> 
			<TrkAttr name="PRODUCTIPADDR" val="cft4"/> 
			<TrkAttr name="PRODUCTOS" val="SYST_UNIX"/> 
			<TrkAttr name="GMTDIFF" val="0"/> 
			<TrkAttr name="RecDate" val="2019.09.03"/> 
			<TrkAttr name="RecTime" val="10:28:21"/> 
			<TrkAttr name="Monitor" val="CFT"/> 
			<TrkAttr name="ReturnMessage" val="transfer aborted                 <IDTU=A000001X PART=CFT_TARGET_1_SSL IDF=0063-TES IDT=I0310282 110 >"/> 
			<TrkAttr name="CycleId" val=""/> 
			<TrkAttr name="IsAlert" val="1"/> 
			<TrkAttr name="ApplicationName" val="CFT_DOCKER_4"/> 
			<TrkAttr name="ApplicationGroup" val="dev.docker"/> 
			<TrkAttr name="IdentMsg" val="CFTT82E"/> 
			<TrkAttr name="Product" val="CFT"/> 
			<TrkAttr name="Monitor" val="XFB"/> 
			<TrkAttr name="EVENTDATE" val="03.09.2019"/> 
			<TrkAttr name="EVENTTIME" val="13:43:26"/> 
			<TrkAttr name="ORIGINDATE" val="01.01.1970"/> 
			</TrkObject> 
	</TrkDescriptor>`

func Exampleconvert1() {
	json, _ := convert1(data)
	fmt.Println(json)
	// Output:
	// {"_type" : "Event","_name" : "XFBTransfer","PRODUCTNAME" : "CFT","PRODUCTIPADDR" : "cft"}
}

func TestXmlFailure(t *testing.T) {
	_, err := convert1("data")
	if err == nil {
		t.Fail()
	}
}

func TestUnexpectedContent(t *testing.T) {
	_, err := convert1("<data/>")
	if err == nil {
		//fmt.Println("error", err)
		t.Fail()
	}
}

func TestConvertToMapISO8859(t *testing.T) {
	_, err := convertToMap(dataISO8859)
	if err != nil {
		fmt.Println("error", err)
		t.Fail()
	}
}

func TestConvertToMapInvalidXML(t *testing.T) {
	_, err := convertToMap(dataInvalidXML)
	if err != nil {
		fmt.Println("error", err)
		t.Fail()
	}
}
