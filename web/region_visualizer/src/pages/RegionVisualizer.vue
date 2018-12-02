<template>
    <div class="container" id="region-visualizer">
        <div id="svg-container">
            <svg>
                <linearGradient v-for="(link, index) in data.links" :key="link.id" :id="'grad-link-' + index" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" :stop-color="link.leftColor.hslaString" />
                    <stop offset="100%" :stop-color="link.rightColor.hslaString" />
                </linearGradient>
                <g id="svg-graph-nodes" stroke="rgba(0,0,0,0.1)">
                    <rect v-for="node in data.nodes" :key="node.id"
                          :x="nodeLeft(node)" :y="node.top" :width="node.right - node.left" :height="node.bottom - node.top"
                          :fill="nodeColor(node)" @mouseover="onNodeHover(node)" @mouseleave="onNodeMouseLeave" @click="onNodeClicked(node)">
                        <text :x="nodeLeft(node)" :y="nodeRight(node)">1</text>
                    </rect>
                </g>
                <g id="svg-region-id-labels">
                    <text v-for="node in data.nodes" :key="node.id"
                          :x="nodeLeft(node) + (node.right - node.left) / 2" :y="(node.top + node.bottom) / 2" text-anchor="middle" dominant-baseline="central"
                          fill="#888888">{{ node.region.id }}</text>
                </g>
                <g id="svg-graph-links" stroke="rgba(0,0,0,0.05)">
                    <path v-for="(link, index) in data.links" :key="link.id"
                          :d="calculatePath(link)" :fill="'url(#grad-link-' + index + ')'" />
                </g>
            </svg>
        </div>
        <vue-slider ref="slider" v-model="displayRange" :interval="1" :processDragable="true" :min="0" :max="100"
                    :tooltipDir="['bottom', 'bottom']" :formatter="'{value}%'" :mergeFormatter="'{value1}% ~ {value2}%'"></vue-slider>
        <div id="svg-node-detail-tip-box" v-if="showNodeInfo">
            <nav class="panel">
                <p class="panel-heading">
                    Node {{ nodeToShow.region.id }}, {{ nodeToShow.timestamp }}
                </p>
                <div class="panel-block">
                    <b-table :data="getNodeDetails(nodeToShow, false)" :columns="[{field: 'name'}, {field: 'value'}]">
                    </b-table>
                </div>
            </nav>
        </div>

        <b-modal :active.sync="isDetailDialogOpen" :width="640" scroll="keep">
            <div class="card">
                <div class="card-content">
                    <div class="content" v-if="selectedNode != null">
                        <h3 title="is-h3">Node {{ selectedNode.timestamp }}, {{ selectedNode.region.id }}</h3>
                        <b-table :data="getNodeDetails(selectedNode, true)" :columns="[{field:'name'},{field:'value'}]">
                        </b-table>
                    </div>
                </div>
            </div>
        </b-modal>
    </div>
</template>
<script lang="ts">
    import { Component, Prop, Vue } from 'vue-property-decorator';
    let vueSlider = require('vue-slider-component');

    import { RegionHistoryNode, RegionHistoryLink } from '../util/RegionHistoryElements'
    import { Color } from '../util/Color';

    @Component({
        components: {
            vueSlider,
        }
    })
    export default class RegionVisualizer extends Vue {
        @Prop(Object) data!: {
            nodes: RegionHistoryNode[],
            links: RegionHistoryLink[],
        };

        displayRange: [number, number] = [0, 100];

        width = 1300;

        hovering = false;
        hoveringNode: RegionHistoryNode | null = null;

        selected = false;
        selectedNode: RegionHistoryNode | null = null;

        isDetailDialogOpen = false;

        get showNodeInfo(): boolean {
            return this.hovering || this.selected;
        }

        get nodeToShow(): RegionHistoryNode {
            if (this.hovering)
                return this.hoveringNode!;
            return this.selectedNode!;
        }

        onNodeHover(node: RegionHistoryNode) {
            this.hoveringNode = node;
            this.hovering = true;
        }

        onNodeMouseLeave() {
            this.hovering = false;
        }

        onNodeClicked(node: RegionHistoryNode) {
            //if (this.selected && node == this.selectedNode) {
            //    this.selected = false;
            //    return;
            //}
            //this.selected = true;
            this.selectedNode = node;
            this.isDetailDialogOpen = true;
        }

        nodeColor(node: RegionHistoryNode): string {
            if (this.isSelected(node) || this.isHovering(node))
                return `hsl(${node.color.hue}, 95%, 70%)`;
            return node.color.hslaString;
        }

        isSelected(node: RegionHistoryNode): boolean {
            return this.selected && node == this.selectedNode;
        }

        isHovering(node: RegionHistoryNode): boolean {
            return this.hovering && node == this.hoveringNode;
        }


        textColor(hue: number): string {
            return `hsl(${hue}, 80%, 40%)`;
        }

        calculatePath(link: RegionHistoryLink): string {
            let left = this.nodeRight(this.data.nodes[link.leftIndex]);
            let right = this.nodeLeft(this.data.nodes[link.rightIndex]);
            let middleLeft = (left * 1 + right) / 2;
            let middleRight = (left + right * 1) / 2;
            let res = `M ${left},${link.topLeft} C ${middleLeft},${link.topLeft} ${middleRight},${link.topRight} ${right},${link.topRight} `
                + `L ${right},${link.bottomRight} C ${middleRight},${link.bottomRight} ${middleLeft},${link.bottomLeft} ${left},${link.bottomLeft} Z`;
            return res;
        }

        getNodeDetails(node: RegionHistoryNode, more: boolean): { name: string, value: string }[] {
            let result = [
                { name: 'Time', value: new Date(node.timestamp / 1000000).toLocaleString() + ", " + node.timestamp % 1000000000 + " ns" },
                { name: 'Event', value: node.eventType },
                { name: 'Start Key', value: node.region.start_key },
                { name: 'End Key', value: node.region.end_key },
            ];

            if (more) {
                let peersStr = "";
                for (let peer of node.region.peers) {
                    if (peersStr.length != 0)
                        peersStr += ", ";
                    peersStr += `{id: ${peer.id}, store_id: ${peer.store_id}}`;
                }

                result = [
                    { name: 'Region Id', value: node.region.id.toString() },
                    { name: 'Time Stamp', value: node.timestamp.toString() }
                ].concat(result).concat([
                    { name: 'Leader', value: node.leaderStoreId.toString() },
                    { name: 'Region Epoch', value: `Ver: ${node.region.region_epoch.version}, Conf: ${node.region.region_epoch.conf_ver}` },
                    { name: 'Peers', value: peersStr }
                ]);
            }


            return result;
        }

        nodeLeft(node: RegionHistoryNode): number {
            return (100 * node.left - this.displayRange[0] * this.width) / (this.displayRange[1] - this.displayRange[0]);
        }

        nodeRight(node: RegionHistoryNode): number {
            return this.nodeLeft(node) + node.right - node.left;
        }
    }
</script>
<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">


    #svg-container {
        width: 100%;
        padding-top: 2px;
    }

        #svg-container > svg {
            width: 100%;
            height: 800px;
            /*border: 1px solid rgba(0, 0, 0, 0.1);*/
        }

            #svg-container > svg * {
                transition: all ease-in-out 0.3s;
            }

    #svg-graph-links > path {
        mix-blend-mode: multiply;
        pointer-events: none;
    }

    #svg-region-id-labels > text {
        mix-blend-mode: multiply;
        pointer-events: none;
    }

    #svg-node-detail-tip-box {
        position: fixed;
        right: 10px;
        bottom: 10px;
        width: auto;
        height: auto;
        opacity: 0.9;
        z-index: 10000;
    }

        #svg-node-detail-tip-box div.panel-block {
            background: white;
        }

    #svg-graph-nodes > rect:hover {
        stroke: rgba(0, 0, 0, 0.5);
    }

    #svg-region-id-labels {
        transition: all ease-in-out 0.3s;
    }
</style>
