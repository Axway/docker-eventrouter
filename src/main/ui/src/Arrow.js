export default function Arrow({ b1, b2, title } = props) {
    if (!b1 || !b2) return <></>
    const pad = 0
    const pad2 = 20

    const relx = -b1.x - b1.width + pad2
    const rely = -Math.min(b1.y + b1.height / 2, b2.y + b2.height / 2) + pad2
    const b1x = Math.round(b1.x + b1.width + relx)
    const b2x = Math.round(b2.x + relx)
    const b1y = Math.round(b1.y + b1.height / 2 + rely)
    const b2y = Math.round(b2.y + b2.height / 2 + rely)
    const bmx = Math.round((b1x + b2x) / 2)

    const path = `M${b1x} ${b1y} C ${bmx} ${b1y} ${bmx} ${b2y} ${b2x} ${b2y}`
    //console.log("arrow", title, b1, b2, path)

    let x = Math.min(b1x, b2x)
    let y = Math.min(b1y, b2y)
    let w = Math.abs(b1x - b2x)
    let h = Math.abs(b1y - b2y)
    let viewBox = `${x - pad2} ${y - pad2} ${w + 2 * pad2} ${h + pad2 * 2}`

    const dx = -w;
    const dy = b1y - b2y > 0 ? 0 : -h;

    return <div className="arrow-container"><svg className="arrow" xmlns="http://www.w3.org/2000/svg"

        viewBox={viewBox}
        style={{ left: dx - pad2, top: dy - pad2, width: (w + 2 * pad2) + "px", height: h + 2 * pad2 + "px" }} >
        <defs>
            <marker
                id='head'
                orient="auto"
                markerWidth='3'
                markerHeight='4'
                refX='1.9'
                refY='2'
            >
                <path d='M0,0 V4 L2,2 Z' fill="black" />
            </marker>
        </defs>

        <path
            id='arrow-line'
            //markerStart='url(#head)'
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
    </div>
}