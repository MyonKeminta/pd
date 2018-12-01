import Axios from 'axios';
import { RawNode } from './RegionHistoryElements';
import { setTimeout } from 'timers';

const USE_MOCK_DATA = false;
function pdPrefix(path: string): string {
    //return "http://localhost:2333/";
    return "/pd/api/v1/history/" + path;
}

const MOCK_DATA = [
    [
        {
            "timestamp": 0,
            "event_type": "fuck",
            "region": {
                "id": 4,
                "start_key": "D",
                "end_key": "F",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [],
            "children": [2]
        },
        {
            "timestamp": 1,
            "event_type": "fuck",
            "region": {
                "id": 1,
                "start_key": "A",
                "end_key": "D",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [],
            "children": [2, 3]
        },
        {
            "timestamp": 9,
            "event_type": "fuck",
            "region": {
                "id": 2,
                "start_key": "C",
                "end_key": "F",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [0, 1],
            "children": [4]
        },
        {
            "timestamp": 10,
            "event_type": "fuck",
            "region": {
                "id": 1,
                "start_key": "A",
                "end_key": "C",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [1],
            "children": [4, 5]
        },
        {
            "timestamp": 13,
            "event_type": "fuck",
            "region": {
                "id": 1,
                "start_key": "B",
                "end_key": "F",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [2, 3],
            "children": [6]
        },
        {
            "timestamp": 15,
            "event_type": "fuck",
            "region": {
                "id": 3,
                "start_key": "A",
                "end_key": "B",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [3],
            "children": []
        },
        {
            "timestamp": 16,
            "event_type": "fuck",
            "region": {
                "id": 1,
                "start_key": "B",
                "end_key": "F",
                "region_epoch": {
                    "conf_ver": 101,
                    "version": 119
                },
                "peers": [
                    {
                        "id": 448632,
                        "store_id": 7
                    },
                    {
                        "id": 664362,
                        "store_id": 167149
                    },
                    {
                        "id": 677615,
                        "store_id": 185553
                    }
                ]
            },
            "leader_store_id": 7,
            "parents": [4],
            "children": []
        }
    ]
]

function getMockData(onSuccess: (_: RawNode[]) => void, _onError: (_: any) => void, onFinish: () => void) {
    setTimeout(() => {
        onSuccess(MOCK_DATA[Math.floor(Math.random() * 1)]);
        onFinish();
    }, 1500);
}

function getDataFromPdApi(onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
    Axios.get<RawNode[]>(pdPrefix("list"))
        .then(res => onSuccess(res.data))
        .catch(onError)
        .then(onFinish);
}

namespace DataSource {
    export function getALlData(onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
        if (USE_MOCK_DATA) {
            getMockData(onSuccess, onError, onFinish);
        } else {
            getDataFromPdApi(onSuccess, onError, onFinish);
        }
    }
}

export default DataSource;
