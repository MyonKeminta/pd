import Axios from 'axios';
import { RawNode } from './RegionHistoryElements';
import { setTimeout } from 'timers';

const USE_MOCK_DATA = false;

function pdPrefix(path: string): string {
    return "/pd/api/v1/history/" + path;
}

const MOCK_DATA = [
    [
        {
            "timestamp": 0,
            "event_type": "Bootstrap",
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
            "leader_store_id": 1,
            "parents": [],
            "children": [1]
        },
        {
            "timestamp": 1,
            "event_type": "LeaderChange",
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
            "leader_store_id": 1,
            "parents": [0],
            "children": [2, 3]
        },
        {
            "timestamp": 9,
            "event_type": "Split",
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
            "leader_store_id": 1,
            "parents": [1],
            "children": [4]
        },
        {
            "timestamp": 9,
            "event_type": "Split",
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
            "leader_store_id": 1,
            "parents": [1],
            "children": [4]
        },
        {
            "timestamp": 13,
            "event_type": "Merge",
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
            "leader_store_id": 1,
            "parents": [2, 3],
            "children": [5]
        },
        {
            "timestamp": 16,
            "event_type": " ",
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
            "leader_store_id": 1,
            "parents": [4],
            "children": []
        }
    ]
]

export type StringDict = { [key: string]: string };

function getMockData(onSuccess: (_: RawNode[]) => void, _onError: (_: any) => void, onFinish: () => void) {
    setTimeout(() => {
        onSuccess(MOCK_DATA[Math.floor(Math.random() * 1)]);
        onFinish();
    }, 1500);
}

function getDataFromPdApi(path: string, params: StringDict, onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
    Axios.get<RawNode[]>(pdPrefix(path), {
        params: params
    })
        .then(res => onSuccess(res.data))
        .catch(onError)
        .then(onFinish);
}

namespace DataSource {
    export function getAllData(params: StringDict, onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
        if (USE_MOCK_DATA) {
            getMockData(onSuccess, onError, onFinish);
        } else {
            getDataFromPdApi("list", params, onSuccess, onError, onFinish);
        }
    }

    export function getDataByRegion(regionId: number, params: StringDict, onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
        if (USE_MOCK_DATA) {
            getMockData(onSuccess, onError, onFinish);
        } else {
            getDataFromPdApi("region/" + regionId, params, onSuccess, onError, onFinish);
        }
    }

    export function getDataByKey(key: string, params: StringDict, onSuccess: (_: RawNode[]) => void, onError: (_: any) => void, onFinish: () => void) {
        if (USE_MOCK_DATA) {
            getMockData(onSuccess, onError, onFinish);
        } else {
            getDataFromPdApi("key/" + key, params, onSuccess, onError, onFinish);
        }
    }
}

export default DataSource;
