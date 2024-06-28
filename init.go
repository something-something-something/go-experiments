package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/indeedeng/iwf-golang-sdk/gen/iwfidl"
	"github.com/indeedeng/iwf-golang-sdk/iwf"
	"io"
	"math/rand"
	"net/http"
	"strconv"
)

//go:embed web/*
var webFiles embed.FS

var registry = iwf.NewRegistry()

type Workflow1 struct {
	iwf.DefaultWorkflowType
}

func NewWorkflow1() iwf.ObjectWorkflow {
	return &Workflow1{}
}

func (workflow Workflow1) GetWorkflowStates() []iwf.StateDef {
	return []iwf.StateDef{
		iwf.StartingStateDef(NewWorkflow1state()),
		iwf.NonStartingStateDef(NewWorkflow2state()),
	}
}

func (workflow Workflow1) GetPersistenceSchema() []iwf.PersistenceFieldDef {
	return []iwf.PersistenceFieldDef{
		iwf.DataAttributeDef("aasomevalue"),
	}
}

func (workflow Workflow1) GetCommunicationSchema() []iwf.CommunicationMethodDef {
	return []iwf.CommunicationMethodDef{
		iwf.SignalChannelDef("Ready"),

		iwf.RPCMethodDef(workflow.rpcThing, nil),
	}
}

type blah struct {
	a string
}

func newBlah() blah {
	var st = ""
	return blah{
		a: st,
	}
}

func (workflow Workflow1) rpcThing(ctx iwf.WorkflowContext, input iwf.Object, persistence iwf.Persistence, communication iwf.Communication) (interface{}, error) {
	return newBlah(), nil

}

type workflow1state struct {
	iwf.WorkflowStateDefaultsNoWaitUntil
}

func NewWorkflow1state() iwf.WorkflowState {
	return &workflow1state{}
}

func (i workflow1state) Execute(ctx iwf.WorkflowContext, input iwf.Object, commandResults iwf.CommandResults, persistance iwf.Persistence, comunication iwf.Communication) (*iwf.StateDecision, error) {
	var myinput = make(map[string]int)
	var persistanceValue = make(map[string]string)
	persistanceValue["something"] = "a value stored"
	persistance.SetDataAttribute("aasomevalue", persistanceValue)
	fmt.Println("workflow executing")
	input.Get(&myinput)
	if myinput["test"] > 0 {
		myinput["test"] = myinput["test"] - 1

		return iwf.MultiNextStatesWithInput(iwf.NewStateMovement(
			workflow1state{},
			myinput,
		), iwf.NewStateMovement(
			workflow1state{},
			myinput,
		), iwf.NewStateMovement(
			workflow1state{},
			myinput,
		), iwf.NewStateMovement(
			workflow1state{},
			myinput,
		)), nil

	} else {
		return iwf.SingleNextState(workflow2state{}, myinput), nil
	}
}

type workflow2state struct {
	iwf.WorkflowStateDefaultsNoWaitUntil
}

func NewWorkflow2state() iwf.WorkflowState {
	return &workflow2state{}
}

func (i workflow2state) Execute(ctx iwf.WorkflowContext, input iwf.Object, commandResults iwf.CommandResults, persistance iwf.Persistence, comunication iwf.Communication) (*iwf.StateDecision, error) {
	var persistanceValue map[string]string
	persistance.GetDataAttribute("aasomevalue", &persistanceValue)
	fmt.Println(persistanceValue["something"])
	fmt.Println("workflowstate2 executing")

	return iwf.GracefulCompletingWorkflow, nil
}

var iwfClient = iwf.NewClient(registry, &iwf.ClientOptions{
	ServerUrl:     "http://iwf:8801",
	WorkerUrl:     "http://127.0.0.1:8803",
	ObjectEncoder: iwf.GetDefaultObjectEncoder(),
})

func main() {

	registry.AddWorkflow(NewWorkflow1())

	var iwfWorker = iwf.NewWorkerService(registry, nil)

	wfservermux := http.NewServeMux()

	wfservermux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {

		io.WriteString(w, "debug")
	})

	wfservermux.HandleFunc(iwf.WorkflowStateWaitUntilApi, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("waituntil")
		var waitUntilReq iwfidl.WorkflowStateWaitUntilRequest
		body, _ := io.ReadAll(r.Body)

		json.Unmarshal(body, &waitUntilReq)
		resp, _ := iwfWorker.HandleWorkflowStateWaitUntil(context.Background(), waitUntilReq)
		j, _ := json.Marshal(resp)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(j))
	})

	wfservermux.HandleFunc(iwf.WorkflowStateExecuteApi, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("exec")
		var Execreq iwfidl.WorkflowStateExecuteRequest
		body, _ := io.ReadAll(r.Body)

		json.Unmarshal(body, &Execreq)
		resp, _ := iwfWorker.HandleWorkflowStateExecute(context.Background(), Execreq)
		j, _ := json.Marshal(resp)
		fmt.Println(string(j))
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(j))
	})

	wfservermux.HandleFunc(iwf.WorkflowWorkerRPCAPI, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("rpc")
		var RPCreq iwfidl.WorkflowWorkerRpcRequest
		body, _ := io.ReadAll(r.Body)

		json.Unmarshal(body, &RPCreq)
		resp, _ := iwfWorker.HandleWorkflowWorkerRPC(context.Background(), RPCreq)
		j, _ := json.Marshal(resp)
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, string(j))
	})
	wfServer := &http.Server{
		Addr:    ":" + iwf.DefaultWorkerPort,
		Handler: wfservermux,
	}

	go func() {
		wfServer.ListenAndServe()
	}()

	fmt.Println("hi")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `
		
		<doctype html>
		<html>
		<head>
			<script type="module" async src="/web/js/test.js"></script>
			<title></title>
		</head>
		<body>
		hello<test-element></test-element>
		</body>
		</html
		`)

	})
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("start")
		var wfid = strconv.FormatUint(rand.Uint64(), 10)

		var myinput = make(map[string]int)
		myinput["test"] = 2

		_, err := iwfClient.StartWorkflow(context.Background(), Workflow1{}, wfid, 300, myinput, nil)
		if err != nil {
			fmt.Println(err)
		}
		w.Header().Set("content-type", "text/html")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `
		
		<doctype html>
		<html>
		<head>
			<script type="module" async src="/web/js/test.js"></script>
			<title></title>
		</head>
		<body>
		start workflow`+wfid+`
		</body>
		</html
		`)

	})
	http.Handle("/web/", http.FileServerFS(webFiles))
	http.ListenAndServe(":8080", nil)
	fmt.Println("text")

}
