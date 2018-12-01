export class Color {
    public hue: number = 0;
    public saturation: number = 0;
    public lightness: number = 0;
    public alpha: number = 1;

    public constructor(h: number, s: number, l: number, a: number) {
        this.hue = h;
        this.saturation = s;
        this.lightness = l;
        this.alpha = a;
    }

    public get hslaString(): string {
        return `hsla(${this.hue}, ${this.saturation * 100}%, ${this.lightness * 100}%, ${this.alpha})`;
    }
}