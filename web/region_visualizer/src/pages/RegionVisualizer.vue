<template>
    <div class="container" id="region-visualizer">
        <div id="svg-container">
            <svg>
                <linearGradient v-for="(link, index) in data.links" :key="link.id" :id="'grad-link-' + index" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" :stop-color="link.leftColor.hslaString" />
                    <stop offset="100%" :stop-color="link.rightColor.hslaString" />
                </linearGradient>
                <g stroke="rgba(0,0,0,0.1)">
                    <rect v-for="node in data.nodes" :key="node.id"
                          :x="node.left" :y="node.top" :width="node.right - node.left" :height="node.bottom - node.top"
                          :fill="node.color.hslaString" @mouseover="onNodeHover(node)" @mouseleave="onNodeMouseLeave" />
                </g>
                <g id="svg-region-id-labels">
                    <text v-for="node in data.nodes" :key="node.id"
                          :x="(node.left + node.right) / 2" :y="(node.top + node.bottom) / 2" text-anchor="middle" dominant-baseline="central"
                          fill="#888888">{{ node.region.id }}</text>
                </g>
                <g id="svg-graph-links" stroke="rgba(0,0,0,0.5)">
                    <path v-for="(link, index) in data.links" :key="link.id"
                          :d="calculatePath(link)" :fill="'url(#grad-link-' + index + ')'" />
                </g>
            </svg>
        </div>
        <transition name="slide-fade">
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
        </transition>
    </div>
</template>

<script lang="ts">
    import { Component, Prop, Vue } from 'vue-property-decorator';

    import { RegionHistoryNode, RegionHistoryLink } from '../util/RegionHistoryElements'

    @Component
    export default class RegionVisualizer extends Vue {
        @Prop(Object) data!: {
            nodes: RegionHistoryNode[],
            links: RegionHistoryLink[],
        };

        hovering = false;
        hoveringNode: RegionHistoryNode | null = null;

        selected = false;
        selectedNode: RegionHistoryNode | null = null;

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


        textColor(hue: number): string {
            return `hsl(${hue}, 80%, 40%)`;
        }

        calculatePath(link: RegionHistoryLink): string {
            let middleLeft = (link.left * 1 + link.right) / 2;
            let middleRight = (link.left + link.right * 1) / 2;
            let res = `M ${link.left},${link.topLeft} C ${middleLeft},${link.topLeft} ${middleRight},${link.topRight} ${link.right},${link.topRight} `
                + `L ${link.right},${link.bottomRight} C ${middleRight},${link.bottomRight} ${middleLeft},${link.bottomLeft} ${link.left},${link.bottomLeft} Z`;
            return res;
        }

        getNodeDetails(node: RegionHistoryNode, more: boolean): { name: string, value: string }[] {
            let result = [
                { name: 'Region Id', value: node.region.id.toString() },
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

                result = result.concat([
                    { name: 'Leader', value: node.leaderStoreId.toString() },
                    { name: 'Region Epoch', value: `Ver: ${node.region.region_epoch.version}, Conf: ${node.region.region_epoch.conf_ver}` },
                    { name: 'Peers', value: peersStr }
                ]);
            }


            return result;
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
            height: 700px;
            border: 1px solid rgba(0, 0, 0, 0.1);
        }

    #svg-container > svg * {
        transition: all ease-in-out 0.5s;
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
    }

    #svg-node-detail-tip-box div.panel-block {
        background: white;
    }
</style>
