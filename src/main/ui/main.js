const { useState, useEffect, useRef, useLayoutEffect } = React

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

function Arrow({ b1, b2, title } = props) {
    /*const [b1, setB1] = useState(0);
    const [b2, setB2] = useState(0);
    useLayoutEffect(() => {
        setB1(ref1.current.getBoundingClientRect())
        setB2(ref2.current.getBoundingClientRect())
    })*/
    if (!b1 || !b2) return <></>
    const b1x = Math.round(b1.x + b1.width)
    const b2x = Math.round(b2.x)
    const b1y = Math.round(b1.y + b1.height / 2)
    const b2y = Math.round(b2.y + b2.height / 2)
    const bmx = Math.round((b1x + b2x) / 2)

    const path = `M${b1x} ${b1y} C ${bmx} ${b1y} ${bmx} ${b2y} ${b2x} ${b2y}`
    //console.log("arrow", title, b1, b2, path)
    const pad = 0
    const pad2 = 20
    let x = Math.min(b1x, b2x)
    let y = Math.min(b1y, b2y)
    let w = Math.abs(b1x - b2x)
    let h = Math.abs(b1y - b2y)
    let viewBox = `${x - pad2} ${y - pad2} ${w + 2 * pad2} ${h + pad2 * 2}`

    if (false) {
        x = 0
        y = 0
        w = Math.max(b1x, b2x)
        h = Math.max(b1y, b2y)
        viewBox = undefined
    }
    return <svg xmlns="http://www.w3.org/2000/svg"
        className="arrow"
        viewBox={viewBox}
        style={{ left: x - pad2, top: y - pad2, width: (w + 2 * pad2) + "px", height: h + 2 * pad2 + "px" }} >
        <defs>
            <marker
                id='head'
                orient="auto"
                markerWidth='3'
                markerHeight='4'
                refX='0.1'
                refY='2'
            >
                <path d='M0,0 V4 L2,2 Z' fill="black" />
            </marker>
        </defs>

        <path
            id='arrow-line'
            markerEnd='url(#head)'
            strokeWidth='4'
            strokeDasharray="10,10"
            strokeDashoffset="0"
            fill='none'
            stroke='black'
            d={path}
        >
            <animate
                attributeName="stroke-dashoffset"
                values="20;0"
                dur="0.5s"
                repeatCount="indefinite" />
        </path>
    </svg>
}

function DisplayProcessor({ x } = props) {
    return <div className="processor">
        <div className="title"><span className="obj-type">Processor</span> {x.Name}</div>
        <div className="activity">out={x.Out} acked={x.Out_ack} in={x.In}</div>
        {/*x.Runtime && <div className="main" onMouseEnter={() => setSideBar(x.Runtime)}>Main</div>*/}
        {x.Runtime && <pre>{JSON.stringify(x.Runtime, null, "  ")}</pre>}
        <div className="all" onMouseEnter={() => setSideBar(x)}>All</div>
        {x.Runtimes && <div className="all" onMouseEnter={() => setSideBar(x.Runtimes)}>Children [{x.Runtimes?.length}]</div>}
        <div>{JSON.stringify(x.cIn)}</div>
    </div>
}

function DisplayStream({ x, processorsD, streamsD, channelsD, pref } = props) {
    const nref = useRef(null);
    const [bbox, setBbox] = useState(null)
    useLayoutEffect(() => {
        const b = nref.current.getBoundingClientRect()
        //console.log("arrow", b)
        setBbox(b)

    }, [])

    const steps = []
    for (let i = 0; i < x.Flow.length; i++) {
        const step = x.Flow[i]
        const processName = x.Name + "/" + step.Name
        const p = processorsD[processName]
        if (i !== 0) {
            steps.push(<span className="">
                <span className="obj-type">Channel</span> {p?.Cin && p.Cin.Name} {p?.Cin && "" + p.Cin.Size}/{p?.Cin && "" + p.Cin.Pos}
            </span>)
        }
        steps.push(<span className="flow-step" key={processName}>
            ({i}) <span className="obj-type">Connector</span> {step.Name}
            {p && <DisplayProcessor x={p} />}
        </span>)
    }

    //console.log(x.Name)
    return <div key={x.Name} className="stream-group">
        <div ref={nref} className="stream">
            <div><span className="obj-type">stream</span> {x.Name}</div>
            <div className="flow-steps">{steps}</div>
        </div>
        {pref && <Arrow b1={pref} b2={bbox} title={"to-" + x.Name} />}
        {x.downstreams && <>
            <div>{"--->"}</div>
            <div className="stream-children">{x.downstreams?.map(x => <DisplayStream x={streamsD[x]} processorsD={processorsD} streamsD={streamsD} channelsD={channelsD} pref={bbox} />)}</div>
        </>}

    </div >
}

function App(props) {
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

    const channels = state.Channels?.Channels.map(x => <li key={x.Name}>{x.Name}</li>)

    const channelsD = {}
    state.Config?.Channels?.forEach((x) => { channelsD[x.Name] = x })

    const processorsD = {}
    state.Processors?.forEach((x) => { const index = x.Flow.Name + "/" + x.FlowStep.Name; processorsD[index] = x; })

    const streamsD = {}
    state.Config?.Streams?.forEach((x) => { streamsD[x.Name] = x })

    //Identify root streams
    const rootStreams = state.Config?.Streams.filter(x => x.Upstream === "")

    const streamsV = rootStreams?.map(x => <div className="stream-line" key={x.Name}><DisplayStream x={x} processorsD={processorsD} streamsD={streamsD} channelsD={channelsD} /></div>)

    return <>
        <h1>QLT Router {props.name}! {status}</h1>
        {JSON.stringify(state.Distribution)}
        <p>channels: [{state.Channels?.Channels.length}] {channels}</p>
        STREAMS: <div className="streams">{streamsV}</div>

        <div className="sidebar"><pre>{JSON.stringify(sidebar, null, "  ")}</pre></div>
    </>;
}

//render(html`<${App} name="World" />`, document.body);
const domContainer = document.querySelector('#content');
const root = ReactDOM.createRoot(domContainer);
root.render(<App />);
