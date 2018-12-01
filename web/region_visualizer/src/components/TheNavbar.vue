<template>
    <nav id="navbar" class="navbar has-shadow" role="navigation" aria-label="main navigation">
        <div class="container">
            <div class="navbar-brand">
                <a class="navbar-item" href="/">这是 Logo</a>

                <a role="button" class="navbar-burger burger" aria-label="menu" aria-expanded="false" data-target="navbarItems">
                    <span aria-hidden="true"></span>
                    <span aria-hidden="true"></span>
                    <span aria-hidden="true"></span>
                </a>
            </div>

            <div class="navbar-menu">
                <div class="navbar-start">
                    <a class="navbar-item" @click.prevent="onAllClicked">
                        <b-icon icon="all-inclusive"></b-icon> <span>All Regions</span>
                    </a>
                </div>
                <div class="navbar-end">
                    <b-dropdown position="is-bottom-left">
                        <a class="navbar-item" slot="trigger">
                            <span>Time Interval: {{}}</span>
                            <b-icon icon="menu-down"></b-icon>
                        </a>
                    </b-dropdown>
                </div>
            </div>
        </div>
    </nav>
</template>

<script lang="ts">
    import { Component, Vue } from 'vue-property-decorator';

    @Component
    export default class Navbar extends Vue {
        useStartTime: boolean = false;
        startDate: Date | null = new Date();

        onAllClicked() {
            this.$emit("on-all-clicked");
        }
    }
</script>

<style scoped>
</style>