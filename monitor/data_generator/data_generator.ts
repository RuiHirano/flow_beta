const fs = require('fs');


const jsonObject = JSON.parse(fs.readFileSync('./../src/sampledata/test.json', 'utf8'));

let movesdata = {}
let initialTime = 0
let time = initialTime


console.log("initialTime", initialTime, initialTime + 1)
jsonObject.data.forEach((stepdata) => {
    stepdata.forEach((agentdata) => {
        let color = [0, 200, 120];
        const id = agentdata.id
        if (movesdata[id] !== undefined) { // 存在する場合
            const data = movesdata[id]
            movesdata[id] = {
                ...data,
                operation: [
                    ...data.operation,
                    {
                        elapsedtime: time,
                        longitude: agentdata.route.position.longitude,
                        latitude: agentdata.route.position.latitude,
                        //color
                    }
                ]
            }
        } else {
            movesdata[id] = {
                //type: id,
                departuretime: time,
                arrivaltime: 10,
                operation: [
                    {
                        elapsedtime: time,
                        longitude: agentdata.route.position.longitude,
                        latitude: agentdata.route.position.latitude,
                        //color
                    }
                ]
            }
        }
    })
    time += 1
});


let result = {
    "timeLength": time - initialTime,
    "timeBegin": 1515800145,
    "bounds": {
        "southlatitude": 35.152210,
        "northlatitude": 35.161499,
        "westlongitiude": 136.971626,
        "eastlongitiude": 136.989379
    },
    "movesbase": Object.values(movesdata)
}

//console.log("data", Object.values(movesdata)[0])
//result.movesbase = Object.values(movesdata)

fs.writeFileSync('./../src/sampledata/output.json', JSON.stringify(result, null, '    '));