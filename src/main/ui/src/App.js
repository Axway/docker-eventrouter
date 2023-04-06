const { useState, useEffect, useRef, useLayoutEffect } = React
import Arrow from "./Arrow.js"
import Graph from "./Graph.js"

async function TextClient(uri) {
    const response = await fetch(uri)
    const body = await response.text()
    return body
}

async function RestClient(uri) {
    const response = await fetch(uri)
    const body = await response.json()
    return body
}

function Tooltip({ obj }) {
    return <div class="tooltip-container">
        <div class="toolip">
            <Pre obj={obj} />
        </div>
    </div>
}

function Pre({ obj }) {
    const a = []
    for (const [k, v] of Object.entries(obj)) {
        if (typeof v !== 'object') {
            a.push((<div className="presection" key={k}><div className="prelabel">{k}</div> <div className="prevalue">{v}</div></div>))
        } else {
            a.push((<div className="presection" key={k}><div className="prelabel">{k}</div> <Pre obj={v} /></div>))
        }
    }
    return <div>{a}</div>
}

function DisplayProcessor({ x } = props) {
    return <div className="processor">
        <div className="title">
            <span className="obj-type">Processor</span> {x.Name}
        </div>
        <div><span className="obj-type">Connector</span> {x.FlowStep.Name}</div>

        <div className="activity"><Pre obj={{ out: x.Out, acked: x.Out_ack, inflight: x.Out - x.Out_ack }} /></div>
        {/*x.Runtime && <div className="main" onMouseEnter={() => setSideBar(x.Runtime)}>Main</div>*/}
        {x.Runtime && <Pre obj={x.Runtime} />}
        <div className="all" onMouseEnter={() => setSideBar(x)}>All</div>
        {<Graph values={historic[processorName(x)]} />}
        {x.Runtimes && <div className="all" onMouseEnter={() => setSideBar(x.Runtimes)}>Children [{x.Runtimes?.length}]</div>}
        <div>{JSON.stringify(x.cIn)}</div>
    </div>
}

function DisplayStream({ x, processorsD, streamsD, channelsD, pref } = props) {
    const nref = useRef(null);
    const [bbox, setBbox] = useState(null)
    useLayoutEffect(() => {
        const b = nref //.current.getBoundingClientRect()
        //console.log("arrow", b)
        setBbox(b)

    }, [])

    const steps = []
    for (let i = 0; i < x.Flow.length; i++) {
        const step = x.Flow[i]
        const processName = x.Name + "/" + step.Name
        const p = processorsD[processName]
        if (i !== 0) {
            steps.push(<span className="channel" key={p.Cin.Name + "-" + i}>
                <span className="obj-type">Channel</span> {p?.Cin && p.Cin.Name} {p?.Cin && "" + p.Cin.Size}/{p?.Cin && "" + p.Cin.Pos}
            </span>)
        }
        steps.push(<span className="flow-step" key={processName + "-" + i}>
            ({i}) <span className="obj-type">Connector</span> {step.Name}
            {p && <DisplayProcessor x={p} />}
        </span>)
    }

    //console.log(x.Name)
    return <div key={x.Name} className="stream-group">
        {pref && <Arrow b1={pref.current.getBoundingClientRect()} b2={bbox.current.getBoundingClientRect()} title={"to-" + x.Name} />}
        <div ref={nref} className="stream">
            <div><span className="obj-type">stream</span> {x.Name}</div>
            <div className="flow-steps">{steps}</div>
        </div>
        {x.downstreams && <>
            <div className="stream-children-arrow">{"--->"}</div>
            <div className="stream-children">{x.downstreams?.map(x => <DisplayStream key={x} x={streamsD[x]} processorsD={processorsD} streamsD={streamsD} channelsD={channelsD} pref={bbox} />)}</div>
        </>}

    </div >
}

const historic = {}
function processorName(x) {
    return x.Flow.Name + "/" + x.FlowStep.Name
}
function addToHistory(name, value) {
    let h = historic[name]
    if (!h) {
        h = historic[name] = []
    }
    h.push(value)
}

export default function App(props) {
    const [status, setStatus] = useState("")
    const [state, setState] = useState({})
    const [sidebar, setSideBar] = useState({})
    const [metrics, setMetrics] = useState("")

    async function fetchState() {

        try {
            const state = await RestClient('/api/state')

            console.log(state)
            //Index Stream/Processors
            const streamsD = {}
            state.Config?.Streams?.forEach((x) => { streamsD[x.Name] = x })

            //For all non root streams, set downstream property
            for (let stream of state.Config?.Streams || []) {
                if (stream.Upstream !== "") {
                    const s = streamsD[stream.Upstream]
                    if (!s) {
                        console.error("NotFound", stream.Upstream)
                        stream.Name = stream.Name + "**BAD Upstream**"
                    } else {
                        if (!s.downstreams) {
                            s.downstreams = []
                        }
                        s.downstreams.push(stream.Name)
                    }
                }
            }

            state.Processors?.forEach((x) => { addToHistory(processorName(x), x.Out) })
            setState(state)
            setStatus("")
        } catch (err) {
            setStatus("ERROR:" + err)
            console.error("", err)
        }
    }

    async function fetchMetrics() {
        const state = await TextClient('/metrics')
        //console.log(state)
        setMetrics(state)
    }

    useEffect(() => {
        fetchState()
        fetchMetrics()
        setInterval(() => {
            fetchState()
            fetchMetrics()
        }, 1000)
    }, [])

    const channels = state.Channels?.Channels.map((x, i) => <li key={x.Name + "-" + i}>{x.Name}</li>)

    const channelsD = {}
    state.Config?.Channels?.forEach((x) => { channelsD[x.Name] = x })

    const processorsD = {}
    state.Processors?.forEach((x) => { processorsD[processorName(x)] = x; })

    const streamsD = {}
    state.Config?.Streams?.forEach((x) => { streamsD[x.Name] = x })

    //Identify root streams
    const rootStreams = state.Config?.Streams.filter(x => x.Upstream === "")

    const streamsV = rootStreams?.map(x => <div className="stream-line" key={x.Name}><DisplayStream x={x} processorsD={processorsD} streamsD={streamsD} channelsD={channelsD} /></div>)
    //console.log("draw")
    return <>
        <h1>QLT Router {props.name}! {status}</h1>
        {JSON.stringify(state.Distribution)}

        <div className="scontent">
            <div className="streams">{streamsV}</div>
        </div>
        <p>channels: [{state.Channels?.Channels.length}] {channels}</p>
        <div className="sidebar"><pre>{JSON.stringify(sidebar, null, "  ")}</pre></div>
    </>;
}

