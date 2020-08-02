
import React, { useEffect, useState } from 'react';
import { HarmoVisLayers, Container, BasedProps, BasedState, connectToHarmowareVis, MovesLayer, Movesbase, MovesbaseOperation, DepotsLayer, DepotsData, LineMapLayer, LineMapData } from 'harmoware-vis';
import io from "socket.io-client";
import { Controller } from '../components';


//const MAPBOX_TOKEN = process.env.REACT_APP_MAPBOX_TOKEN ? process.env.REACT_APP_MAPBOX_TOKEN : "";
const MAPBOX_TOKEN = 'pk.eyJ1IjoicnVpaGlyYW5vIiwiYSI6ImNqdmc0bXJ0dTAzZDYzem5vMmk0ejQ0engifQ.3k045idIb4JNvawjppzqZA'


class Harmoware extends Container<BasedProps & BasedState> {
    render() {
        const { actions, depotsData, viewport, movesbase } = this.props;
        //console.log("test2", movesbase)
        return (<HarmowarePage {...this.props} />)
    }
}

const socket: SocketIOClient.Socket = io();

interface AreaInfo {
    Id: string
    Name: string
    ControlArea: Coord[]
    DuplicateArea: Coord[]
}

interface Coord {
    Lat: number
    Lon: number
}

const HarmowarePage: React.FC<BasedProps & BasedState> = (props) => {
    const { actions, depotsData, viewport, movesbase, movedData, routePaths, clickedObject } = props
    //console.log("test1", movesbase)
    const [linedata, setLinedata] = useState<LineMapData[]>([])
    const [areadata, setAreadata] = useState<AreaInfo[]>([])
    const [movesdata, setMovesdata] = useState<Movesbase[]>([])
    //const movesdata = [...movesbase]

    const getAgents = (data: any) => {
        const time = Date.now() / 1000; // set time as now. (If data have time, ..)
        const newMovesbase: Movesbase[] = [];
        // useEffect内では外側のstateは初期化時のままなので、set関数内で過去のstateを取得する必要がある
        setMovesdata((movesdata) => {
            //console.log("socketData: ", movesdata);
            data.forEach((value: any) => {
                const { mtype, id, lat, lon } = JSON.parse(
                    value
                );
                let color = [0, 200, 120];

                let isExist = false;
                // operation内のelapsedtimeなどのオブジェクトは2つ以上ないと表示されないので注意

                movesdata.forEach((movedata) => {
                    //console.log("id, type: ", id, movedata.type)
                    if (id === movedata.type) {
                        //console.log("match")
                        // 存在する場合、更新
                        newMovesbase.push({
                            ...movedata,
                            operation: [
                                ...movedata.operation,
                                {
                                    elapsedtime: time,
                                    position: [lon, lat, 0],
                                    color
                                }
                            ]
                        });
                        isExist = true
                    }
                })

                if (!isExist) {
                    // 存在しない場合、新規作成
                    let color = [0, 255, 0];
                    newMovesbase.push({
                        type: id,
                        operation: [
                            {
                                elapsedtime: time,
                                position: [lon, lat, 0],
                                color
                            }
                        ]
                    });
                }


            });
            return newMovesbase
        })

        actions.updateMovesBase(newMovesbase);
    }

    const getAreas = (data: any) => {
        console.log("areaInfo", data);

        const linedata: LineMapData[] = []
        const areas = convertJsonToArea(data)
        setAreadata(areas)

        areas.forEach((areaInfo: AreaInfo) => {
            const { maxLat, maxLon, minLat, minLon } = getCoordRange(areaInfo.ControlArea)
            linedata.push({
                "sourcePosition": [minLon, minLat, 0],
                "targetPosition": [minLon, maxLat, 0]
            })
            linedata.push({
                "sourcePosition": [minLon, maxLat, 0],
                "targetPosition": [maxLon, maxLat, 0]
            })
            linedata.push({
                "sourcePosition": [maxLon, maxLat, 0],
                "targetPosition": [maxLon, minLat, 0]
            })
            linedata.push({
                "sourcePosition": [maxLon, minLat, 0],
                "targetPosition": [minLon, minLat, 0]
            })
        })

        setLinedata(linedata)
    }

    useEffect(() => {
        socket.on("agents", (data: any) => getAgents(data));
        socket.on("areas", (data: any) => getAreas(data));

        //testAgent()

        console.log(process.env);
        if (actions) {
            actions.setViewport({
                ...props.viewport,
                longitude: 136.9831702,
                latitude: 35.1562909,
                width: window.screen.width,
                height: window.screen.height,
                zoom: 16
            })
            actions.setSecPerHour(3600);
            actions.setLeading(2)
            actions.setTrailing(5)

        }
    }, [])


    //console.log("render: ", viewport, actions)
    return (
        <div>
            <Controller {...props} />
            <HarmoVisLayers
                viewport={viewport} actions={actions}
                mapboxApiAccessToken={MAPBOX_TOKEN}
                layers={[
                    new LineMapLayer({
                        data: linedata,
                        getWidth: (x) => 10,
                    }),
                    new MovesLayer({
                        routePaths,
                        movesbase,
                        movedData,
                        clickedObject,
                        actions,
                        optionVisible: false,
                        //lightSettings,
                        //layerRadiusScale: 0.1,
                        getRadius: x => 0.5,
                        //getRouteWidth: x => 1,
                        //optionCellSize: 2,
                        //sizeScale: 1,
                        iconChange: false,
                        //optionChange: false, // this.state.optionChange,
                        //onHover
                    }),

                ]}
            />
        </div>
    );
}



async function timeout(ms: number) {
    await new Promise(resolve => setTimeout(resolve, ms));
    return
}

const getCoordRange = ((coords: Coord[]) => {
    let maxLat = Number.NEGATIVE_INFINITY
    let maxLon = Number.NEGATIVE_INFINITY
    let minLat = Number.POSITIVE_INFINITY
    let minLon = Number.POSITIVE_INFINITY

    coords.forEach((coord) => {
        if (coord.Lat > maxLat) {
            maxLat = coord.Lat
        }
        if (coord.Lon > maxLon) {
            maxLon = coord.Lon
        }
        if (coord.Lat < minLat) {
            minLat = coord.Lat
        }
        if (coord.Lon < minLon) {
            minLon = coord.Lon
        }
    })

    return { maxLat, maxLon, minLat, minLon }
})

const convertJsonToArea = ((data: any[]) => {
    const areas: AreaInfo[] = []
    data.forEach((areaStr: any) => {
        const areaJson = JSON.parse(areaStr);
        var area: AreaInfo = { ControlArea: [], DuplicateArea: [], Name: "", Id: "" }
        areaJson.control_area.forEach((arg: any) => {
            area.ControlArea.push({ Lat: arg.latitude, Lon: arg.longitude })
        })
        areaJson.duplicate_area.forEach((arg: any) => {
            area.DuplicateArea.push({ Lat: arg.latitude, Lon: arg.longitude })
        })
        areaJson.id ? area.Id = areaJson.id : area.Id = ""
        areaJson.name ? area.Name = areaJson.name : area.Name = ""
        areas.push(area)
    })

    return areas
})

export default connectToHarmowareVis(Harmoware);