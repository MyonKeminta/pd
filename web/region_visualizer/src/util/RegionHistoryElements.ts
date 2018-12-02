import { Color } from "@/util/Color";

export class Peer {
    public id: number = 0;
    public store_id: number = 0;
}

export class Region {
    public id: number = 0;
    public start_key: string = "";
    public end_key: string = "";
    public region_epoch: { version: number, conf_ver: number } = { version: 0, conf_ver: 0 };
    public peers: Peer[] = [];
}

export class RegionHistoryNode {
    public id: string = "";
    // Region properties
    public timestamp: number = 0;
    public eventType: string = "";
    public region: Region = new Region();
    public leaderStoreId: number = 0;

    // Properties of the graph
    public left: number = 0;
    public right: number = 0;
    public top: number = 0;
    public bottom: number = 0;
    public color: Color = new Color(0, 0, 0, 1);

    public leftIndices: number[] = [];
    public rightIndices: number[] = [];
    public leftLinkIndices: number[] = [];
    public rightLinkIndices: number[] = [];
}

export class RegionHistoryLink {
    public leftIndex: number = 0;
    public rightIndex: number = 0;

    public id: string = "";
    public left: number = 0;
    public right: number = 0;
    public topLeft: number = 0;
    public topRight: number = 0;
    public bottomLeft: number = 0;
    public bottomRight: number = 0;

    public leftColor: Color = new Color(0, 0, 0, 1);
    public rightColor: Color = new Color(0, 0, 0, 1);
}

export class RawNode {
    // Region properties
    public timestamp: number = 0;
    public event_type: string = "";
    public region: Region = new Region();
    public leader_store_id: number = 0;
    public parents: number[] = [];
    public children: number[] = [];
}

/*
 * Return value format of API:
 * [
 *   {
 *     timestamp: number,
 *     region: Region,
 *     eventType: string,
 *     leaderStoreId: number,
 *     startKey: string,
 *     endKey: string,
 *     parents: number[],
 *   },
 *   ...
 * ]
 **/

namespace GraphicsConfig {
    export const nodeWidth = 25;
    export const nodeMaxHeight = 1500;
    export const nodeVerticalGap = 10;

    export const nodeSaturation = 0.95;
    export const nodeLightness = 0.85;
    export const linkSaturation = 0.95;
    export const linkLightness = 0.8;
    export const linkAlpha = 0.3;
}

function scaleTime(interval: number): number {
    return Math.log(interval+ 2);
}

export function generateNodePositions(nodes: RegionHistoryNode[], maxNodeHeight:number, width: number, height: number) {
    // Calculate X position
    let totalTime = 0;
    let timeIntervals: number[] = [0];
    for (let i = 1; i < nodes.length; ++i) {
        let left = nodes[i - 1];
        let right = nodes[i];
        if (left.timestamp == right.timestamp) {
            continue;
        }
        totalTime += scaleTime(right.timestamp - left.timestamp);
        timeIntervals.push(totalTime);
    }

    let scaleX = (width - GraphicsConfig.nodeWidth * timeIntervals.length) / totalTime;
    nodes[0].left = 0;
    nodes[0].right = nodes[0].left + GraphicsConfig.nodeWidth;
    let timeIntervalIndex = 0;
    for (let i = 1; i < nodes.length; ++i) {
        let node = nodes[i];
        if (node.timestamp != nodes[i - 1].timestamp) {
            ++timeIntervalIndex;
        }
        node.left = timeIntervals[timeIntervalIndex] * scaleX + timeIntervalIndex * GraphicsConfig.nodeWidth;
        node.right = node.left + GraphicsConfig.nodeWidth;
    }

    // Calculate Y position
    // keyOrder: [(key, nodeIndex, isStartKey, orderIndex), ...]
    let keyOrder: { key: string, nodeIndex: number, isStartKey: boolean, orderIndex: number }[] = [];
    for (let i = 0; i < nodes.length; ++i) {
        let node = nodes[i];
        keyOrder.push({ key: node.region.start_key, nodeIndex: i, isStartKey: true, orderIndex: 0 });
        keyOrder.push({ key: node.region.end_key, nodeIndex: i, isStartKey: false, orderIndex: 0 });
    }
    keyOrder.sort((lhs, rhs): number => {
        if (!lhs.isStartKey && lhs.key.length == 0) {
            if (!rhs.isStartKey && rhs.key.length == 0)
                return 0;
            return 1;
        } else if (!rhs.isStartKey && rhs.key.length == 0) {
            return -1;
        }
        if (lhs.key < rhs.key)
            return -1;
        else if (lhs.key > rhs.key)
            return 1;
        return 0;
    });
    let uniqueKeyCount = 0;
    for (let i = 1; i < keyOrder.length; ++i) {
        if (keyOrder[i].key != keyOrder[i - 1].key ||
            (keyOrder[i].isStartKey != keyOrder[i - 1].isStartKey && keyOrder[i].key.length == 0 && keyOrder[i - 1].key.length == 0)) {
            ++uniqueKeyCount;
        }
        keyOrder[i].orderIndex = uniqueKeyCount;
    }
    ++uniqueKeyCount;
    for (let item of keyOrder) {
        let pos = height * (item.orderIndex + 1.0) / (uniqueKeyCount + 1);
        let node = nodes[item.nodeIndex];
        if (item.isStartKey) {
            node.top = pos + GraphicsConfig.nodeVerticalGap / 2;
        } else {
            node.bottom = pos - GraphicsConfig.nodeVerticalGap / 2;
            // When we get to there, node.top must have already been set.
            // So truncate it's height here.
            if (node.bottom - node.top > maxNodeHeight) {
                let middle = (node.bottom + node.top) / 2;
                node.top = middle - maxNodeHeight / 2;
                node.bottom = middle + maxNodeHeight / 2;
            }
        }
    }
}

export function generateLinkPosition(nodes: RegionHistoryNode[], links: RegionHistoryLink[]) {
    for (let node of nodes) {
        let nodeHeight = node.bottom - node.top;

        let leftLinkCount = node.leftLinkIndices.length;
        let leftLinkHeight = nodeHeight / leftLinkCount;
        for (let i = 0; i < leftLinkCount; ++i) {
            let link = links[node.leftLinkIndices[i]];
            link.right = node.left;
            link.topRight = node.top + leftLinkHeight * i;
            link.bottomRight = node.top + leftLinkHeight * i + leftLinkHeight;
        }

        let rightLinkCount = node.rightLinkIndices.length;
        let rightLinkHeight = nodeHeight / rightLinkCount;
        for (let i = 0; i < rightLinkCount; ++i) {
            let link = links[node.rightLinkIndices[i]];
            link.left = node.right;
            link.topLeft = node.top + rightLinkHeight * i;
            link.bottomLeft = node.top + rightLinkHeight * i + rightLinkHeight;
        }
    }
}

export function generateElementColor(nodes: RegionHistoryNode[], links: RegionHistoryLink[], random: boolean) {
    if (random) {
        for (let node of nodes) {
            node.color = new Color(
                Math.random() * 270,
                GraphicsConfig.nodeSaturation,
                GraphicsConfig.nodeLightness,
                1
            );
        }
    } else {
        let differentStoresCount = 0;
        let storeSet: any = {};
        for (let node of nodes) {
            if (storeSet[node.leaderStoreId] == undefined) {
                storeSet[node.leaderStoreId] = differentStoresCount;
                ++differentStoresCount;
            }
        }
        // Avoid too hight contrast of two oppisite color
        let colorCount = Math.max(differentStoresCount, 5);
        for (let node of nodes) {
            let hue = 90 + storeSet[node.leaderStoreId] * 270 / colorCount;
            if (hue > 360)
                hue -= 360;
            node.color = new Color(
                hue,
                GraphicsConfig.nodeSaturation,
                GraphicsConfig.nodeLightness,
                1
            );
        }
    }

    for (let link of links) {
        link.leftColor = new Color(
            nodes[link.leftIndex].color.hue,
            GraphicsConfig.linkSaturation,
            GraphicsConfig.linkLightness,
            GraphicsConfig.linkAlpha
        );
        link.rightColor = new Color(
            nodes[link.rightIndex].color.hue,
            GraphicsConfig.linkSaturation,
            GraphicsConfig.linkLightness,
            GraphicsConfig.linkAlpha
        );
    }
}

export function generateFromRawNode(rawNodes: RawNode[], maxNodeHeight: number, width: number, height: number, randColor: boolean): { nodes: RegionHistoryNode[], links: RegionHistoryLink[] } {
    let nodes: RegionHistoryNode[] = [];
    let links: RegionHistoryLink[] = [];

    // Load nodes
    for (let i = 0; i < rawNodes.length; ++i) {
        let rawNode = rawNodes[i];
        let node = new RegionHistoryNode();
        node.timestamp = rawNode.timestamp;
        node.eventType = rawNode.event_type;
        node.leaderStoreId = rawNode.leader_store_id;
        node.region = rawNode.region;
        node.id = `node-${node.timestamp}-${node.region.id}`;

        node.leftIndices = rawNode.parents;
        node.rightIndices = rawNode.children;
        nodes.push(node);

        for (let j = 0; j < rawNode.parents.length; ++j) {
            let link = new RegionHistoryLink();
            link.leftIndex = rawNode.parents[j];
            link.rightIndex = i;
            link.id = `link-${nodes[link.leftIndex].id}-${node.id}`;

            links.push(link);
        }
    }

    // Connect nodes to links
    links.sort((lhs, rhs) => {
        let lhsLeftKey = nodes[lhs.leftIndex].region.start_key;
        let rhsLeftKey = nodes[rhs.leftIndex].region.start_key;
        if (lhsLeftKey < rhsLeftKey)
            return -1;
        if (lhsLeftKey > rhsLeftKey)
            return 1;

        let lhsRightKey = nodes[lhs.rightIndex].region.start_key;
        let rhsRightKey = nodes[rhs.rightIndex].region.start_key;
        if (lhsRightKey < rhsRightKey)
            return -1;
        if (lhsRightKey > rhsRightKey)
            return 1;

        return 0;
    });
    for (let i = 0; i < links.length; ++i) {
        let link = links[i];
        nodes[link.leftIndex].rightLinkIndices.push(i);
        nodes[link.rightIndex].leftLinkIndices.push(i);
    }

    generateNodePositions(nodes, maxNodeHeight, width, height);
    generateLinkPosition(nodes, links);
    generateElementColor(nodes, links, randColor);

    return {
        nodes: nodes,
        links: links,
    }
}