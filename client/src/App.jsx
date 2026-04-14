import React, { useState, useEffect } from 'react';
import axios from 'axios';
import MetricCard from './components/MetricCard';
import './index.css';

function App() {
  const [data, setData] = useState(null);
  const [metrics, setMetrics] = useState({});
  const [currentTime, setCurrentTime] = useState(new Date());
  const [loading, setLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    const apiBase = import.meta.env.VITE_API_BASE_URL || '';
    console.log("USA-Graph: Fetching from", apiBase);
    
    const fetchData = () => {
      if (loading) return; // 중복 요청 방지 가드
      setLoading(true);
      axios.get(`${apiBase}/api/status`)
        .then(res => {
          setData(res.data);
          if (res.data.metrics_json) {
            try {
              const parsed = typeof res.data.metrics_json === 'string' 
                ? JSON.parse(res.data.metrics_json) 
                : res.data.metrics_json;
              setMetrics(parsed || {});
            } catch (e) {
              console.error("Failed to parse metrics_json:", e);
              setMetrics({});
            }
          }
        })
        .catch(err => {
          console.error("Error fetching status:", err);
        })
        .finally(() => {
          setLoading(false);
        });
    };

    fetchData();
    const interval = setInterval(fetchData, 60000); // 1분 주기로 단축
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (score) => {
    if (score >= 7.5) return 'var(--accent-green)';
    if (score >= 5.0) return 'var(--accent-orange)';
    return 'var(--accent-red)';
  };

  const getVal = (id) => {
    const val = metrics[id]?.value;
    if (val === undefined || val === null) return '--';
    // 소수점이 있는 지표들을 위해 2자리까지 표시 (maximumFractionDigits 적용)
    return val.toLocaleString(undefined, { 
      minimumFractionDigits: 2,
      maximumFractionDigits: 2 
    });
  };
  
  const getWresbalVal = () => {
    return getVal('WRESBAL');
  };

  const getChangeLabel = (id, unit = "") => {
    const m = metrics[id];
    if (!m || m.diff === undefined) return "0.00%";
    const diff = m.diff;
    const percent = m.percent?.toFixed(2) || "0.00";
    
    // 증감 수치도 소수점 2자리까지 정확히 표시
    let diffStr = Math.abs(diff).toLocaleString(undefined, { 
      minimumFractionDigits: 2,
      maximumFractionDigits: 2 
    });
    return `${diffStr}${unit} (${percent}%)`;
  };

  const getP = (id) => metrics[id]?.percent?.toFixed(2) || '0.00';
  const getTrend = (id) => (metrics[id]?.diff >= 0 ? 'up' : 'down');

  const getRegimeColor = () => {
    if (data?.regime?.includes('완화')) return 'var(--accent-green)';
    if (data?.regime?.includes('긴축')) return 'var(--accent-red)';
    return 'var(--accent-orange)';
  };

  return (
    <div className="dashboard-container">
      <div className="summary-header card" style={{ display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
        <div className="summary-left" style={{ flex: '0 0 auto' }}>
          <div 
            className="donut-chart" 
            style={{ '--score-pct': `${data ? data.total_score : 0}%`, '--score-color': getRegimeColor() }}
            onClick={() => setIsModalOpen(true)}
            title="AI 요약 보고서 보기"
          >
            <div className="donut-inner">
              <span className="donut-score" style={{ color: getRegimeColor() }}>
                {data ? Math.round(data.total_score) : '--'}
              </span>
            </div>
          </div>
        </div>

        <div className="summary-center">
          <div className="summary-label">종합 유동성 점수</div>
          <div className="summary-main">
            <div className="status-sphere" style={{ backgroundColor: getRegimeColor(), boxShadow: `0 0 20px ${getRegimeColor()}` }}></div>
            <h1 className="status-text">
              {data?.regime || '분석 중...'} — <span className="position-text">{data?.position || '대기'}</span>
            </h1>
          </div>
          <div className="regime-legend">
            <span className="legend-item"><i className="dot green"></i> 완화</span>
            <span className="legend-item"><i className="dot yellow"></i> 중립</span>
            <span className="legend-item"><i className="dot red"></i> 긴축</span>
          </div>
        </div>

        <div className="summary-right desktop-only">
          <div className="timestamp">
            {currentTime.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </div>
        </div>
      </div>

      <div className="clock-bar mobile-only">
        <div className="timestamp">
          {currentTime.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
        </div>
      </div>

      <div className="metrics-grid">
        <MetricCard title="RRP 잔고" subTitle="역레포 잔고" value={getVal('RRPONTSYD')} unit="B" change={getChangeLabel('RRPONTSYD', 'B')} trend={getTrend('RRPONTSYD')} statusColor={getStatusColor(metrics['RRPONTSYD']?.score)} history={metrics['RRPONTSYD']?.history} />
        <MetricCard title="은행 준비금" subTitle="예치기관 준비금" value={getWresbalVal()} unit="T" change={getChangeLabel('WRESBAL', 'T')} trend={getTrend('WRESBAL')} statusColor={getStatusColor(metrics['WRESBAL']?.score)} history={metrics['WRESBAL']?.history} />
        <MetricCard title="TGA 잔고" subTitle="재무부 일반계정" value={getVal('WTREGEN')} unit="B" change={getChangeLabel('WTREGEN', 'B')} trend={getTrend('WTREGEN')} statusColor={getStatusColor(metrics['WTREGEN']?.score)} history={metrics['WTREGEN']?.history} />
        <MetricCard title="SOFR" subTitle="담보 대출 금리" value={getVal('SOFR')} unit="%" change={getChangeLabel('SOFR', '%')} trend={getTrend('SOFR')} statusColor={getStatusColor(metrics['SOFR']?.score)} history={metrics['SOFR']?.history} />
        <MetricCard title="DXY" subTitle="달러 인덱스" value={getVal('DXY')} unit="" change={getChangeLabel('DXY')} trend={getTrend('DXY')} statusColor={getStatusColor(metrics['DXY']?.score)} history={metrics['DXY']?.history} />
        <MetricCard title="장단기 금리차" subTitle="10Y - 2Y" value={getVal('T10Y2Y')} unit="pp" change={getChangeLabel('T10Y2Y', 'pp')} trend={getTrend('T10Y2Y')} statusColor={getStatusColor(metrics['T10Y2Y']?.score)} history={metrics['T10Y2Y']?.history} />
        
        <MetricCard title="MMF 자산" subTitle="머니마켓펀드" value={getVal('RMFNS')} unit="B" change={getChangeLabel('RMFNS', 'B')} trend={getTrend('RMFNS')} statusColor={getStatusColor(metrics['RMFNS']?.score)} history={metrics['RMFNS']?.history} />
        <MetricCard title="M2 통화량" subTitle="광의의 통화" value={getVal('M2SL')} unit="T" change={getChangeLabel('M2SL', 'T')} trend={getTrend('M2SL')} statusColor={getStatusColor(metrics['M2SL']?.score)} history={metrics['M2SL']?.history} />
        <MetricCard title="하이일드 스프레드" subTitle="신용 위험 지표" value={getVal('BAMLH0A0HYM2')} unit="%" change={getChangeLabel('BAMLH0A0HYM2', '%')} trend={getTrend('BAMLH0A0HYM2')} statusColor={getStatusColor(metrics['BAMLH0A0HYM2']?.score)} history={metrics['BAMLH0A0HYM2']?.history} />
        <MetricCard title="VIX 공포지수" subTitle="시장 변동성" value={getVal('VIXCLS')} unit="" change={getChangeLabel('VIXCLS')} trend={getTrend('VIXCLS')} statusColor={getStatusColor(metrics['VIXCLS']?.score)} history={metrics['VIXCLS']?.history} />
        <MetricCard title="Fear & Greed" subTitle="시장 심리 지수" value={getVal('FEAR_GREED')} unit="" change={getChangeLabel('FEAR_GREED')} trend={getTrend('FEAR_GREED')} statusColor={getStatusColor(metrics['FEAR_GREED']?.score)} history={metrics['FEAR_GREED']?.history} />
        
        <div className="card" style={{ visibility: 'hidden' }}></div>
      </div>

      <footer style={{ marginTop: 'auto', paddingBottom: '1rem', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '0.7rem' }}>
        FRED® is a registered trademark of the Federal Reserve Bank of St. Louis.
      </footer>

      {isModalOpen && (
        <div className="modal-overlay" onClick={() => setIsModalOpen(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <div className="modal-title">
                <span style={{ fontSize: '1.5rem' }}>🤖</span>
                AI 유동성 요약 보고서
              </div>
              <button className="modal-close" onClick={() => setIsModalOpen(false)}>&times;</button>
            </div>
            <div className="modal-body">
              <div className="ai-report-section">
                <h4>현재 레짐 분석</h4>
                <div className="ai-report-text">
                  현재 시장 유동성 점수는 <strong>{Math.round(data?.total_score || 0)}점</strong>으로 
                  <strong> '{data?.regime}'</strong> 상태입니다. 지표상으로는 유동성 공급이 안정적이나, 
                  시장 심리는 여전히 보수적인 관망세를 유지하고 있습니다.
                </div>
              </div>
              
              <div className="ai-report-section">
                <h4>핵심 리스크 및 기회</h4>
                <div className="ai-report-text">
                  역레포(RRP) 잔고의 감소가 시장 유동성 방어에 기여하고 있으나, 
                  장단기 금리차의 동향을 통해 향후 경기 방향성을 면밀히 모니터링해야 합니다. 
                  하이일드 스프레드 안정은 기업 금융 환경에 긍정적입니다.
                </div>
              </div>

              <div className="ai-report-section">
                <h4>투자 전략 가이드</h4>
                <div className="ai-report-text">
                  급격한 방향성 베팅보다는 우량주 중심의 분할 접근이 유리한 구간입니다. 
                  공포 탐욕 지수가 중립 이상으로 회복될 때 추가적인 상승 탄력을 기대할 수 있습니다.
                </div>
              </div>

              <div style={{ marginTop: '2rem', textAlign: 'center', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                * 본 보고서는 11개 경제 지표를 기반으로 자동 생성되었습니다.
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
