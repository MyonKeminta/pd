<template>
    <div class="container" id="region-visualizer">
        <div id="svg-container">
            <svg>
                <linearGradient v-for="(link, index) in data.links" :key="link.id" :id="'grad-link-' + index" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" :stop-color="link.leftColor.hslaString" />
                    <stop offset="100%" :stop-color="link.rightColor.hslaString"  />
                </linearGradient>
                <g stroke="rgba(0,0,0,0.3)">
                    <rect v-for="node in data.nodes" :key="node.id"
                          :x="node.left" :y="node.top" :width="node.right - node.left" :height="node.bottom - node.top"
                          :fill="node.color.hslaString" />
                </g>
                <g id="svg-region-id-labels">
                    <text v-for="node in data.nodes" :key="node.id"
                          :x="(node.left + node.right) / 2" :y="(node.top + node.bottom) / 2" text-anchor="middle" dominant-baseline="central" 
                          fill="#888888"
                          >{{ node.region.id }}</text>
                </g>
                <g id="svg-graph-links">
                    <path v-for="(link, index) in data.links" :key="link.id" 
                          :d="calculatePath(link)" :fill="'url(#grad-link-' + index + ')'" />
                </g>
            </svg>
        </div>
    </div>
</template>

<script lang="ts">
    import { Component, Prop, Vue } from 'vue-property-decorator';

    import {RegionHistoryNode, RegionHistoryLink} from '../util/RegionHistoryElements'

    @Component
    export default class RegionVisualizer extends Vue {
        @Prop(Object) data!: {
            nodes: RegionHistoryNode[],
            links: RegionHistoryLink[],
        };

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

    #svg-graph-links > path {
        mix-blend-mode: multiply;
    }

    #svg-region-id-labels > text {
        mix-blend-mode: multiply;
    }
</style>
