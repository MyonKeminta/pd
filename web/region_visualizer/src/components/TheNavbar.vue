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
                 <div class="navbar-item">
                     <b-input name="navbar-input-key" width="10" v-model="key"></b-input>
                    <a class="navbar-item" @click.prevent="onKeyClicked">
                        <b-icon icon="all-inclusive"></b-icon> <span>Query Key</span>
                    </a>
                </div>

                <div class="navbar-end">
                    <b-dropdown position="is-bottom-left">
                        <a class="navbar-item" slot="trigger">
                            <span>{{ timeIntervalString }}</span>
                            <b-icon icon="menu-down"></b-icon>
                        </a>

                        <b-dropdown-item custom paddingless id="time-interval-drop-down">
                            <div class="modal-card" style="width:auto">
                                <section class="modal-card-body" id="time-interval-form-fields">
                                    <section>
                                        <div class="field">
                                            <b-switch v-model="editedUseStartTime">Limit Start Time</b-switch>
                                        </div>
                                        <div class="field">
                                            <b-field label="Start Time" v-show="editedUseStartTime">
                                                <b-datepicker placeholder="Start Date" icon="calendar-today" rounded v-model="editedStartTime"></b-datepicker>
                                            </b-field>
                                        </div>
                                        <div class="field">
                                            <b-field v-show="editedUseStartTime">
                                                <b-timepicker rounded placeholder="Start Time" icon="clock" v-model="editedStartTime"></b-timepicker>
                                            </b-field>
                                        </div>

                                    </section>

                                    <section>
                                        <div class="field">
                                            <b-switch v-model="editedUseEndTime">Limit End Time</b-switch>
                                        </div>
                                        <div class="field">
                                            <b-field label="End Time" v-show="editedUseEndTime">
                                                <b-datepicker placeholder="End Date" icon="calendar-today" rounded v-model="editedEndTime"></b-datepicker>
                                            </b-field>
                                        </div>
                                        <div class="field">
                                            <b-field v-show="editedUseEndTime">
                                                <b-timepicker rounded placeholder="End Time" icon="clock" v-model="editedEndTime"></b-timepicker>
                                            </b-field>
                                        </div>
                                    </section>
                                </section>

                                <footer class="modal-card-foot">
                                    <button class="button is-primary" @click="onTimeRangeOkClick">Ok</button>
                                </footer>
                            </div>
                        </b-dropdown-item>
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
        startTime: Date = new Date();
        useEndTime: boolean = false;
        endTime: Date = new Date();

        editedUseStartTime: boolean = false;
        editedStartTime: Date = new Date();
        editedUseEndTime: boolean = false;
        editedEndTime: Date = new Date();

        key: string = "";

        onAllClicked() {
            this.$emit("on-all-clicked");
        }

        mounted() {
            this.startTime.setDate(this.startTime.getDate() - 1);
        }

        onTimeRangeOkClick() {
            this.useStartTime = this.editedUseStartTime;
            this.startTime = this.editedStartTime;
            this.useEndTime = this.editedUseEndTime;
            this.endTime = this.editedEndTime;

            this.$emit("on-time-range-set", this.useStartTime ? this.startTime : null, this.useEndTime ? this.endTime : null);
        }

        get timeIntervalString(): string {
            if (this.useStartTime) {
                if (this.useEndTime) {
                    return this.startTime.toLocaleString() + " to " + this.endTime.toLocaleString();
                } else {
                    return "After " + this.startTime.toLocaleString();
                }
            } else {
                if (this.useEndTime) {
                    return "Before " + this.endTime.toLocaleString();
                } else {
                    return "Time range not set";
                }
            }
        }

        onKeyClicked() {
            this.$emit("on-key-clicked", this.key);
        }
    }
</script>

<style scoped>
    #time-interval-drop-down * {
        overflow: visible;
    }

    #time-interval-form-fields > section {
        padding-bottom: 9px;
    }
</style>