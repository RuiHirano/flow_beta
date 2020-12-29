
import React, { useEffect, useState } from 'react';
import { HarmoVisLayers, Container, BasedProps, BasedState, connectToHarmowareVis, MovesLayer, Movesbase, MovesbaseOperation, DepotsLayer, DepotsData, LineMapLayer, LineMapData } from 'harmoware-vis';
import { Controller } from '../components';
import Dropzone from 'react-dropzone'
import { setMovesBase } from 'harmoware-vis/lib/src/actions';
import { StaticMap } from 'react-map-gl';
import moment, { Moment } from "moment"
const DeckGL = require('@deck.gl/react');
const { ScatterplotLayer } = require('@deck.gl/layers');

//const MAPBOX_TOKEN = process.env.REACT_APP_MAPBOX_TOKEN ? process.env.REACT_APP_MAPBOX_TOKEN : "";
const MAPBOX_TOKEN = 'pk.eyJ1IjoicnVpaGlyYW5vIiwiYSI6ImNqdmc0bXJ0dTAzZDYzem5vMmk0ejQ0engifQ.3k045idIb4JNvawjppzqZA'

class Harmoware extends Container<BasedProps & BasedState> {
    render() {
        const { actions, depotsData, viewport, movesbase } = this.props;
        //console.log("test2", movesbase)
        return (<HarmowarePage {...this.props} />)
    }
}

const createData = (data: any) => {
    console.log(data);
    let time = Math.floor(Date.now() / 1000)
    console.log("time: ", time, Date.now())
    let color = [0, 200, 120];
    const newMovesbase: any = []
    data.forEach((stepdata: any) => {  // step毎のデータ
        const stepMovesbase = [...newMovesbase]
        console.log("test2", stepMovesbase.length)
        time = Math.floor(time + 1)
        console.log("time: ", time)
        stepdata.forEach((agentdata: any) => {   // agentデータ
            let isExist = false
            stepMovesbase.forEach((movebase: any, index: number) => {
                //console.log("test", movebase.type, agentdata.id, movebase.type === agentdata.id)
                if (movebase.type === agentdata.id) {
                    //console.log("test2", movebase.type === agentdata.id)
                    isExist = true
                    newMovesbase[index] = {
                        ...movebase,
                        operation: [
                            ...movebase.operation,
                            {
                                elapsedtime: time,
                                position: [agentdata.route.position.longitude, agentdata.route.position.latitude, 0],
                                color
                            }
                        ]
                    };
                }
            });
            if (!isExist) {
                // 存在しない場合、新規作成
                let color = [0, 255, 0];
                newMovesbase.push({
                    type: agentdata.id,
                    operation: [
                        {
                            elapsedtime: time,
                            position: [agentdata.route.position.longitude, agentdata.route.position.latitude, 0],
                            color
                        }
                    ]
                });
            }

        });
    });
    console.log("newMovesbase", newMovesbase)
    return newMovesbase
}


const HarmowarePage: React.FC<BasedProps & BasedState> = (props) => {

    const { actions, depotsData, viewport, movesbase, movedData, routePaths, clickedObject } = props

    const [movesdata, setMovesdata] = useState<Movesbase[]>([])

    const pickFile = (file: File) => {
        var fileReader = new FileReader();
        fileReader.onload = function () {
            if (typeof fileReader.result === "string") {
                const data = JSON.parse(fileReader.result)
                //const newMovesbase = createData(data)
                //actions.updateMovesBase(newMovesbase)
                runDataLoop(data)
                //actions.updateMovesBase(data)
            }
        }
        fileReader.readAsText(file);
        console.log(JSON.stringify(file))
    }

    const createData2 = (stepdata: any) => {
        let time = Math.floor(Date.now() / 1000)
        console.log("time: ", time, Date.now())
        let color = [0, 200, 120];
        const newMovesbase: any = []
        setMovesdata((movesdata) => {
            stepdata.forEach((agentdata: any) => {   // agentデータ
                let isExist = false
                movesdata.forEach((movedata: any, index: number) => {
                    //console.log("test", movedata.type, agentdata.id, movedata.type === agentdata.id)
                    if (movedata.type === agentdata.id) {
                        //console.log("test2", movedata.type === agentdata.id)
                        isExist = true
                        newMovesbase[index] = {
                            ...movedata,
                            operation: [
                                ...movedata.operation,
                                {
                                    elapsedtime: time,
                                    position: [agentdata.route.position.longitude, agentdata.route.position.latitude, 0],
                                    color
                                }
                            ]
                        };
                    }
                });
                if (!isExist) {
                    // 存在しない場合、新規作成
                    let color = [0, 255, 0];
                    newMovesbase.push({
                        type: agentdata.id,
                        operation: [
                            {
                                elapsedtime: time,
                                position: [agentdata.route.position.longitude, agentdata.route.position.latitude, 0],
                                color
                            }
                        ]
                    });
                }

            });
            return newMovesbase
        })
        console.log("newMovesbase", newMovesbase)
        return newMovesbase
    }

    const runDataLoop = async (data: any) => {
        for (let i = 0; i < data.length; i++) {
            const stepData = data[i]
            const newMovesbase = createData2(stepData)
            actions.updateMovesBase(newMovesbase);
            await timeout(1000)
        }
    }

    useEffect(() => {

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


    return (
        <div>

            <div>
                <Dropzone onDrop={acceptedFiles => pickFile(acceptedFiles[0])}>
                    {({ getRootProps, getInputProps }) => (
                        <section>
                            <div style={{ height: 100 }} {...getRootProps()}>
                                <input {...getInputProps()} />
                                <p>Drag 'n' drop some files here, or click to select files</p>
                            </div>
                        </section>
                    )}
                </Dropzone>
            </div>
            <div style={{ height: '90%' }}>
                <Controller {...props} />
                <HarmoVisLayers
                    viewport={viewport} actions={actions}
                    mapboxApiAccessToken={MAPBOX_TOKEN}
                    layers={[
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
        </div>
    );
}

/*interface Coord {
    Lat: number
    Lon: number
}

interface RouteBase {
    time: Moment
    latitude: number
    logitude: number
}

interface Agent {
    id: string
    color: number[]
    radius: number
    route: RouteBase[]
}

interface TimeStep {
    time: Moment
    agents: Agent[]
}

class Manager {
    agents: { [id: string]: Agent }
    history: { [time: number]: TimeStep }
    globalTime: Moment
    startFlg: boolean

    constructor() {
        this.agents = {}
        this.history = []
        this.globalTime = moment()
    }

    async start() {
        this.startFlg = true
        while (this.startFlg) {
            this.globalTime.add(1, "seconds")
            await timeout(1000)
        }
    }

    stop() {
        this.startFlg = false
    }

    setAgent(agent: Agent) {
        this.agents[agent.id] = agent
    }

    getAgents() {
        return this.history[this.globalTime.unix()]
    }
}*/




async function timeout(ms: number) {
    await new Promise(resolve => setTimeout(resolve, ms));
    return
}

export default connectToHarmowareVis(Harmoware);