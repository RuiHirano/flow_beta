import React, { useCallback, useEffect, useState } from 'react';
import { Typography } from "@material-ui/core"
import {
    ResponsiveContainer, LineChart, Line, XAxis, YAxis, ReferenceLine, ReferenceArea,
    ReferenceDot, Tooltip, CartesianGrid, Legend, Brush, ErrorBar, AreaChart, Area,
    Label, LabelList
} from 'recharts';

const mockdata: GraphData[] = [
    { timestamp: 1, susceptible: 1000, infected: 2400, recovered: 3400 },
    { timestamp: 24, susceptible: 300, infected: 4567, recovered: 2400 },
    { timestamp: 35, susceptible: 280, infected: 1398, recovered: 2400 },
    { timestamp: 47, susceptible: 200, infected: 9800, recovered: 2400 },
    { timestamp: 5, susceptible: 278, infected: 5000, recovered: 2400 },
    { timestamp: 6, susceptible: 189, infected: 4800, recovered: 2400 },
    { timestamp: 7, susceptible: 189, infected: 4800, recovered: 2400 },
    { timestamp: 8, susceptible: 189, infected: 4800, recovered: 2400 },
    { timestamp: 9, susceptible: 189, infected: 4800, recovered: 2400 },
    { timestamp: 10, susceptible: 189, infected: 4800, recovered: 2400 },
    { timestamp: 1, susceptible: 1000, infected: 2400, recovered: 3400 },
    { timestamp: 24, susceptible: 300, infected: 4567, recovered: 2400 },
    { timestamp: 35, susceptible: 280, infected: 1398, recovered: 2400 },
    { timestamp: 47, susceptible: 200, infected: 9800, recovered: 2400 },
];

export type GraphData = { timestamp: number, susceptible: number, infected: number, recovered: number }
export const useStatsGraph = () => {
    const [data, setData] = useState<GraphData[]>([])

    const addGraphData = (graphData: GraphData) => {
        setData([...data, graphData])
    }
    const renderStatsGraphView = useCallback(() => {
        return (
            <div style={{ zIndex: 100, position: "fixed", bottom: 10 }}>
                <Typography>感染者数推移</Typography>
                <LineChart
                    width={400}
                    height={400}
                    data={data}
                    margin={{ top: 5, right: 20, left: 10, bottom: 5 }}
                >
                    <XAxis dataKey="timestamp" />
                    <Tooltip label={"test"} />
                    <CartesianGrid stroke="#f5f5f5" />
                    <Line type="monotone" dataKey="susceptible" stroke="#00ff00" yAxisId={0} strokeWidth={3} />
                    <Line type="monotone" dataKey="infected" stroke="#ff1493" yAxisId={0} strokeWidth={3} />
                    <Line type="monotone" dataKey="recovered" stroke="#00bfff" yAxisId={0} strokeWidth={3} />
                </LineChart>
            </div>
        )
    }, [data])

    return { "renderStatsGraphView": renderStatsGraphView, "setGraphData": setData }
}
