
function aMin(a) {
    return a.reduce((p, x) => x < p ? x : p, a[0])
}

function aMax(a) {
    return a.reduce((p, x) => x > p ? x : p, a[0])
}

function assert(a, b, msg) {
    if (a !== b) {
        throw new Error("Assertion failed: " + msg + ':' + a + "!=" + b)
    }
}

assert(aMin([1, 3, 4]), 1)
assert(aMin([4, -1, 3]), -1)
assert(aMin([3, 4, 1]), 1)
assert(aMax([1, 3, 4]), 4)
assert(aMax([4, 1, 3]), 4)
assert(aMax([1, 3, 4]), 4)

export default function Graph({ values }) {
    const width = 100
    const height = 50
    const vals = values.filter((x, i) => i > values.length - width)
    const vals2 = vals.filter((x, i) => i != 0).map((x, i) => x - vals[i])


    const min = aMin(vals2)
    const max = aMax(vals2)

    if (min === max) {
        return <></>
    }

    const path = vals2.map((x, i) => " " + i + "," + Math.round(height - (x - min) * height / (max - min)))

    return <div>min={min} max={max}
        <svg className="graph" xmlns="http://www.w3.org/2000/svg"
            viewBox={`0 0 ${width} ${height}`}
            style={{ width: width + "px", height: height + "px" }}
        >

            <polyline
                id='arrow-line'
                //markerStart='url(#head)'
                //markerEnd='url(#head)' 
                //strokeWidth='4'
                //strokeDasharray="10,10"
                //strokeDashoffset="0"
                fill='none'
                stroke='black'
                points={path}
            />
        </svg>
    </div>
}
