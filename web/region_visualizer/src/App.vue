<template>
    <div id="app">
        <Navbar @on-all-clicked="onAllClicked" @on-time-range-set="onTimeRangeSet"/>
        <div class="container">
            <!--<h1 class="title is-1">Region History Visualizer</h1>-->
            <RegionVisualizer :data="data" />
        </div>
        <b-loading :is-full-page="true" :active.sync="requestInProgress" :can-cancel="false"></b-loading>
    </div>
</template>

<script lang="ts">
    import { Component, Vue } from 'vue-property-decorator';
    import Navbar from './components/TheNavbar.vue';
    import RegionVisualizer from './pages/RegionVisualizer.vue';

    import { RegionHistoryNode, RegionHistoryLink, RawNode, generateFromRawNode } from './util/RegionHistoryElements';
    import DataSource from './util/DataSource';

    @Component({
        components: {
            Navbar,
            RegionVisualizer,
        }
    })
    export default class App extends Vue {
        data: {
            nodes: RegionHistoryNode[],
            links: RegionHistoryLink[],
        } = { nodes: [], links: [] };

        startTime: Date | null = null;
        endTime: Date | null = null;

        prevRequest: string = "";

        requestInProgress = false;

        prepareRequestNewData() {
            if (this.requestInProgress) {
                throw "Request in progress. Cancelled.";
            }
            this.requestInProgress = true;
        }

        receiveNewData(rawNodes: RawNode[]) {
            let res = generateFromRawNode(rawNodes, 1000, 700);
            this.data.nodes = res.nodes;
            this.data.links = res.links;
        }

        handleRequestDataError(err: any) {
            alert(err);
        }

        finishRequestData() {
            this.requestInProgress = false;
        }

        refresh() {
            if (this.prevRequest == "all") {
                this.onAllClicked();
            }
        }

        // Navbar click logic
        onAllClicked() {
            this.prevRequest = "all";
            this.prepareRequestNewData();
            DataSource.getALlData(this.receiveNewData, this.handleRequestDataError, this.finishRequestData);
        }


        onTimeRangeSet(startTime: Date | null, endTime: Date | null) {
            this.startTime = startTime;
            this.endTime = endTime;

            this.refresh();
        }
    }
</script>

<style lang="scss">
</style>
