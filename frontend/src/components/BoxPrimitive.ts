import type {
  ISeriesPrimitive,
  SeriesAttachedParameter,
  Time,
  IPrimitivePaneView,
  IPrimitivePaneRenderer
} from 'lightweight-charts';

class BoxRenderer implements IPrimitivePaneRenderer {
  private _x1: number | null;
  private _x2: number | null;
  private _y1: number | null;
  private _y2: number | null;

  constructor(x1: number | null, x2: number | null, y1: number | null, y2: number | null) {
    this._x1 = x1;
    this._x2 = x2;
    this._y1 = y1;
    this._y2 = y2;
  }

  draw(target: any) {
    if (
      this._x1 === null ||
      this._x2 === null ||
      this._y1 === null ||
      this._y2 === null ||
      isNaN(this._x1) ||
      isNaN(this._x2) ||
      isNaN(this._y1) ||
      isNaN(this._y2)
    ) {
      return;
    }

    target.useMediaCoordinateSpace((scope: any) => {
      const ctx = scope.context;
      ctx.beginPath();
      
      const x = Math.min(this._x1!, this._x2!);
      const y = Math.min(this._y1!, this._y2!);
      const width = Math.abs(this._x1! - this._x2!);
      const height = Math.abs(this._y1! - this._y2!);

      ctx.rect(x, y, width, height);

      // Semi-transparent purple fill
      ctx.fillStyle = 'rgba(147, 51, 234, 0.15)'; 
      ctx.fill();

      // Sleek solid purple border
      ctx.strokeStyle = 'rgba(168, 85, 247, 0.7)'; 
      ctx.lineWidth = 1.5;
      ctx.setLineDash([4, 4]); // Dashed border for high premium visual look
      ctx.stroke();
      ctx.setLineDash([]); // Reset line dash
    });
  }
}

class BoxPaneView implements IPrimitivePaneView {
  private _source: BoxPrimitive;

  constructor(source: BoxPrimitive) {
    this._source = source;
  }

  renderer(): IPrimitivePaneRenderer {
    return this._source.getRenderer();
  }
}

export class BoxPrimitive implements ISeriesPrimitive<Time> {
  private _chart: any = null;
  private _series: any = null;
  private _paneViews: BoxPaneView[];

  private _startTime: number;
  private _endTime: number;
  private _minPrice: number;
  private _maxPrice: number;

  constructor(startTime: number, endTime: number, minPrice: number, maxPrice: number) {
    this._startTime = startTime;
    this._endTime = endTime;
    this._minPrice = minPrice;
    this._maxPrice = maxPrice;
    this._paneViews = [new BoxPaneView(this)];
  }

  attached(param: SeriesAttachedParameter<Time>) {
    this._chart = param.chart;
    this._series = param.series;
  }

  detached() {
    this._chart = null;
    this._series = null;
  }

  paneViews() {
    return this._paneViews;
  }

  getRenderer(): IPrimitivePaneRenderer {
    if (!this._chart || !this._series) {
      return new BoxRenderer(null, null, null, null);
    }

    const timeScale = this._chart.timeScale();
    const x1 = timeScale.timeToCoordinate(this._startTime as Time);
    const x2 = timeScale.timeToCoordinate(this._endTime as Time);
    const y1 = this._series.priceToCoordinate(this._minPrice);
    const y2 = this._series.priceToCoordinate(this._maxPrice);

    return new BoxRenderer(x1, x2, y1, y2);
  }

  updateAllViews() {}
}
