<template>
    <div id="app">
        <Navbar @on-all-clicked="onAllClicked" />
        <div class="container">
            <!--<h1 class="title is-1">Region History Visualizer</h1>-->
            <RegionVisualizer :data="data" />
        </div>
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

        hasQuery = false;

        receiveNewData(rawNodes: RawNode[]) {
            let res = generateFromRawNode(rawNodes, 1000, 700);
            this.data.nodes = res.nodes;
            this.data.links = res.links;
        }

        handleRequestDataError(err: any) {
            alert(err);
        }

        // Navbar click logic
        onAllClicked() {
            DataSource.getALlData(this.receiveNewData, this.handleRequestDataError, () => { });
        }
        
    }
</script>

<style>
</style>
