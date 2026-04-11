import React from 'react';
import { Line } from 'react-chartjs-2';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Filler,
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Filler
);

const MiniChart = ({ data, color }) => {
  const chartData = {
    labels: data.map((_, i) => i),
    datasets: [
      {
        data: data,
        borderColor: color,
        borderWidth: 1.5, /* 선 굵기 살짝 조정 */
        pointRadius: 0,
        tension: 0, /* 곡선 제거하여 각진 모양으로 변경 */
        fill: true,
        backgroundColor: (context) => {
          const ctx = context.chart.ctx;
          const gradient = ctx.createLinearGradient(0, 0, 0, 80);
          gradient.addColorStop(0, color.replace('1)', '0.15)'));
          gradient.addColorStop(1, 'transparent');
          return gradient;
        },
      },
    ],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { display: false }, tooltip: { enabled: false } },
    scales: {
      x: { display: false },
      y: { display: false },
    },
  };

  return (
    <div style={{ height: '55px', width: '100%', marginTop: 'auto' }}>
      <Line data={chartData} options={options} />
    </div>
  );
};

const MetricCard = ({ title, subTitle, value, change, trend, unit, statusColor, history = [] }) => {
  const isUp = trend === 'up';
  
  // 히스토리 데이터가 있는 경우 실제 지동 반영, 없으면 비워둠
  // 마지막 데이터 포인트로 현재 실시간 value 추가
  const currentValNum = parseFloat(value.replace(/,/g, ''));
  const chartDataPoints = history.length > 0 ? [...history, currentValNum] : [];
  
  const trendColor = isUp ? '#00ff88' : '#ff4d4d';

  return (
    <div className="card metric-card">
      <div className="card-header">
        <div className="title-group">
          <h3 className="card-title">{title}</h3>
          <p className="card-subtitle">{subTitle}</p>
        </div>
        <div className="status-dot" style={{ backgroundColor: statusColor, boxShadow: `0 0 8px ${statusColor}` }}></div>
      </div>
      
      <div className="card-body">
        <div className="value-group">
          <span className="current-value">{value}<span className="unit-text">{unit}</span></span>
          <div className="change-group" style={{ color: trendColor }}>
            <span className="trend-arrow">{isUp ? '▲' : '▼'}</span>
            <span className="change-value">{change}</span>
          </div>
        </div>
      </div>

      {chartDataPoints.length > 0 ? (
        <MiniChart data={chartDataPoints} color={trendColor} />
      ) : (
        <div style={{ height: '55px', width: '100%', marginTop: 'auto', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '0.6rem', color: '#666' }}>
          No graph data
        </div>
      )}
    </div>
  );
};

export default MetricCard;
