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
  const [aiReport, setAiReport] = useState(null);
  const [isAiLoading, setIsAiLoading] = useState(false);

  useEffect(() => {
    const timer = setInterval(() => setCurrentTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    if (isModalOpen && !aiReport && !isAiLoading) {
      const apiBase = import.meta.env.VITE_API_BASE_URL || '';
      setIsAiLoading(true);
      axios.get(`${apiBase}/api/report`)
        .then(res => {
          setAiReport(res.data.cached_content);
        })
        .catch(err => {
          console.error("AI Report fetch failed:", err);
          setAiReport("보고서 데이터를 가져오는 데 실패했습니다.");
        })
        .finally(() => {
          setIsAiLoading(false);
        });
    }
  }, [isModalOpen]);

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
            <div className="modal-body" style={{ minHeight: '200px' }}>
              <div style={{ paddingBottom: '1rem', borderBottom: '1px solid var(--border-color)', marginBottom: '1.5rem' }}>
                <span style={{ color: 'var(--text-secondary)', fontSize: '0.9rem' }}>대시보드 실시간 점수 동기화</span><br/>
                <strong style={{ fontSize: '1.2rem', color: getRegimeColor() }}>{Math.round(data?.total_score || 0)}점 ({data?.regime})</strong>
              </div>

              {isAiLoading ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '150px', color: 'var(--text-secondary)' }}>
                  <div className="spinner" style={{ marginBottom: '1rem', width: '30px', height: '30px', border: '3px solid var(--border-color)', borderTopColor: 'var(--accent-orange)', borderRadius: '50%', animation: 'spin 1s linear infinite' }}></div>
                  <div>AI 요약 보고서를 생성 중입니다...</div>
                </div>
              ) : (
                <div className="ai-report-text" style={{ whiteSpace: 'pre-wrap', lineHeight: '1.8' }}>
                  {aiReport ? aiReport.split('\n').map((line, i) => {
                    if (line.startsWith('###')) return <h4 key={i} style={{ color: 'var(--accent-orange)', marginTop: '1.5rem', marginBottom: '0.5rem' }}>{line.replace(/^#+\s*/, '')}</h4>;
                    if (line.startsWith('**') && line.endsWith('**')) return <strong key={i} style={{ display: 'block', marginTop: '1rem' }}>{line.replace(/\*\*/g, '')}</strong>;
                    if (line.trim() === '') return <br key={i} />;
                    // 간단한 볼드 처리
                    const formattedLine = line.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
                    return <div key={i} dangerouslySetInnerHTML={{ __html: formattedLine }} />;
                  }) : '보고서 데이터가 없습니다.'}
                </div>
              )}

              <div style={{ marginTop: '2.5rem', textAlign: 'center', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                * 본 보고서는 11개 경제 지표를 기반으로 자동 생성되었습니다.
                <style>{`@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }`}</style>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
