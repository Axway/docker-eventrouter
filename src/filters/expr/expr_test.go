package expr

import (
	"testing"

	"axway.com/qlt-router/src/log"
	"github.com/expr-lang/expr"
)

var qltmsg1 = `<?xml version="1.0" encoding="UTF-8"?>
    <TrkDescriptor> 
		<TrkXML VERSION="1.0"/> 
		<TrkObject> 
			<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/> 
			<TrkAttr name="PRODUCTNAME" val="CFT"/> 
			<TrkAttr name="PRODUCTIPADDR" val="cft1"/> 
			<TrkAttr name="PRODUCTOS" val="SYST_UNIX"/> 
			<TrkAttr name="CycleId" val=""/> 
			<TrkAttr name="IsAlert" val="1"/> 
			<TrkAttr name="ApplicationName" val="CFT_DOCKER_1"/> 
			<TrkAttr name="ApplicationGroup" val="dev.cft.docker"/> 
			<TrkAttr name="Product" val="CFT"/> 
			<TrkAttr name="Monitor" val="XFB"/> 
		</TrkObject> 
	</TrkDescriptor>`

var qltmsg2 = `<?xml version="1.0" encoding="UTF-8"?>
		<TrkDescriptor> 
			<TrkXML VERSION="1.0"/> 
			<TrkObject> 
				<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/> 
				<TrkAttr name="PRODUCTNAME" val="CFT"/> 
				<TrkAttr name="PRODUCTIPADDR" val="cft2"/> 
				<TrkAttr name="PRODUCTOS" val="SYST_UNIX"/> 
				<TrkAttr name="CycleId" val=""/> 
				<TrkAttr name="IsAlert" val="1"/> 
				<TrkAttr name="ApplicationName" val="CFT_1"/> 
				<TrkAttr name="ApplicationGroup" val="dev.cft"/> 
				<TrkAttr name="Product" val="CFT"/> 
				<TrkAttr name="Monitor" val="XFB"/> 
			</TrkObject> 
		</TrkDescriptor>`

var qltmsg3 = `<?xml version="1.0" encoding="UTF-8"?>
		<TrkDescriptor> 
			<TrkXML VERSION="1.0"/> 
			<TrkObject> 
				<TrkIdentifier TYPE="Event" NAME="XFBTransfer" VERSION="1.0"/> 
				<TrkAttr name="PRODUCTNAME" val="ST"/> 
				<TrkAttr name="PRODUCTIPADDR" val="st"/> 
				<TrkAttr name="PRODUCTOS" val="SYST_WIN"/> 
				<TrkAttr name="CycleId" val=""/> 
				<TrkAttr name="IsAlert" val="1"/> 
				<TrkAttr name="ApplicationName" val="ST_1"/> 
				<TrkAttr name="ApplicationGroup" val="dev.st"/> 
				<TrkAttr name="Product" val="ST"/> 
				<TrkAttr name="Monitor" val="XFB"/> 
			</TrkObject> 
		</TrkDescriptor>`

var jsonmsg1 = `{"qlttype": "Event" , "qltname": "XFBTransfer" , 
"productname": "CFT", "productos": "SYST_UNIX", "direction": "S", 
"state": "COMPLETED", "receiverid": "PARIS", "senderid": "NEWYORK", 
"originalsenderid": "NEWYORK", "finalreceiverid": "PARIS", 
"machine": "CFT-UNIX", "monitor": "CFT"}`

var jsonmsg2 = `{"qlttype": "Event" , "qltname": "XFBLog" , 
"productname": "CFT", "productos": "SYST_WIN", "monitor": "CFT", 
"returnmessage": "The Transfer CFT license has expired", "isalert": "1", 
"applicationname": "ITEM-4B7WCY3.FTOVO", "product": "CFT"}`

var jsonmsg3 = `{"qlttype": "Event" , "qltname": "XFBLog" , 
"productname": "ST", "productos": "SYST_WIN", "monitor": "ST", 
"returnmessage": "Dummy message", "isalert": "1", "product": "ST"}`

func TestQLTFilter1(t *testing.T) {
	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		//expr.AsBool(),
	}
	code := `msg.productname == "CFT"`
	ctxS := "TestQLTFilter1"

	program, err := expr.Compile(code, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "code:", code, "error", err)
		t.Fail()
	}

	output, _ := filter(ctxS, program, qltmsg1)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg2)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg3)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
}

func TestQLTFilter2(t *testing.T) {
	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}
	code := `msg.applicationname startsWith "CFT"`
	ctxS := "TestQLTFilter2"

	program, err := expr.Compile(code, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "code:", code, "error", err)
		t.Fail()
	}

	output, _ := filter(ctxS, program, qltmsg1)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg2)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg3)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
}

func TestQLTFilter3(t *testing.T) {
	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}
	code := `msg.applicationgroup endsWith "st"`
	ctxS := "TestQLTFilter3"

	program, err := expr.Compile(code, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "code:", code, "error", err)
		t.Fail()
	}

	output, _ := filter(ctxS, program, qltmsg1)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg2)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, qltmsg3)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
}

func TestJsonFilter1(t *testing.T) {
	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}
	code := `msg.originalsenderid != "" ? msg.originalsenderid == msg.senderid : false`
	ctxS := "TestJsonFilter1"

	program, err := expr.Compile(code, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "code:", code, "error", err)
		t.Fail()
	}

	output, _ := filter(ctxS, program, jsonmsg1)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, jsonmsg2)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, jsonmsg3)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
}

func TestJsonFilter2(t *testing.T) {
	env := map[string]map[string]string{}
	options := []expr.Option{
		expr.Env(env),
		expr.AllowUndefinedVariables(), // Allow the use of undefined variables.
		expr.AsBool(),
	}
	code := `msg.productos contains "WIN"`
	ctxS := "TestJsonFilter2"

	program, err := expr.Compile(code, options...)
	if err != nil {
		log.Errorc(ctxS, "Expression compiling failed... ", "code:", code, "error", err)
		t.Fail()
	}

	output, _ := filter(ctxS, program, jsonmsg1)
	if output == true {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "false", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, jsonmsg2)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
	output, _ = filter(ctxS, program, jsonmsg3)
	if output == false {
		log.Errorc(ctxS, "Wrongly matched", "expected:", "true", "got", output)
		t.Fail()
	}
}
