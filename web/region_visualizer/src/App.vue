<template>
    <div id="app">
        <Navbar @on-all-clicked="onAllClicked" @on-key-clicked="onKeyClicked" @on-region-clicked="onRegionClicked" @on-time-range-set="onTimeRangeSet" @on-settings-changed="onSettingsChanged" />
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

    import { RegionHistoryNode, RegionHistoryLink, RawNode, generateFromRawNode, generateLinkPosition, generateNodePositions, generateElementColor } from './util/RegionHistoryElements';
    import DataSource, { StringDict } from './util/DataSource';

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
        prevKey: string = "";
        prevRegionId: number = 0;

        requestInProgress = false;

        nodeFullHeight = false;
        nodeRandomColor = false;

        prepareRequestNewData() {
            if (this.requestInProgress) {
                throw "Request in progress. Cancelled.";
            }
            this.requestInProgress = true;
        }

        receiveNewData(rawNodes: RawNode[]) {
            let res = generateFromRawNode(
                rawNodes, this.nodeFullHeight ? 10000 : 150,
                1300,
                700,
                this.nodeRandomColor
            );
            this.data.nodes = res.nodes;
            this.data.links = res.links;
        }

        handleRequestDataError(err: any) {
            this.$toast.open({
                message: 'Failed getting data: ' + err,
                type: 'is-danger',
                position: 'is-bottom-right',
            });
        }

        finishRequestData() {
            this.requestInProgress = false;
        }

        refresh() {
            if (this.prevRequest == "all") {
                this.requestAllData();
            } else if (this.prevRequest == "key") {
                this.requestDataByKey(this.prevKey);
            }
        }

        createRequestParam(): StringDict {
            let result: StringDict = {};
            if (this.startTime != null) {
                result["start"] = (this.startTime.getTime() * 1000000).toString(); // To nanos
            }
            if (this.endTime != null) {
                result["end"] = (this.endTime.getTime() * 1000000).toString(); // To nanos
            }
            return result;
        }

        requestAllData() {
            this.prepareRequestNewData();
            this.prevRequest = "all";
            DataSource.getAllData(this.createRequestParam(), this.receiveNewData, this.handleRequestDataError, this.finishRequestData);
        }

        requestDataByKey(key: string) {
            this.prepareRequestNewData();
            this.prevRequest = "key";
            this.prevKey = key;
            DataSource.getDataByKey(key, this.createRequestParam(), this.receiveNewData, this.handleRequestDataError, this.finishRequestData);
        }

        requestDataByRegionId(regionId: number) {
            this.prepareRequestNewData();
            this.prevRequest = "region";
            this.prevRegionId = regionId;
            DataSource.getDataByRegion(regionId, this.createRequestParam(), this.receiveNewData, this.handleRequestDataError, this.finishRequestData);
        }

        // Navbar click logic
        onAllClicked() {
            this.requestAllData();
        }

        onKeyClicked(key: string) {
            this.requestDataByKey(key);
        }

        onRegionClicked(regionId: number) {
            this.requestDataByRegionId(regionId);
        }

        onTimeRangeSet(startTime: Date | null, endTime: Date | null) {
            this.startTime = startTime;
            this.endTime = endTime;

            this.refresh();
        }

        onSettingsChanged(name: string, value: boolean) {
            if (name == "fullHeight") {
                this.nodeFullHeight = value;
                generateNodePositions(this.data.nodes, value ? 10000 : 150, 1300, 700);
                generateLinkPosition(this.data.nodes, this.data.links);
            } else if (name == "randomColor") {
                this.nodeRandomColor = value;
                generateElementColor(this.data.nodes, this.data.links, this.nodeRandomColor);
            }
        }
    }
</script>
<style lang="scss">
</style>
