import React, { useState, useEffect, useRef, useCallback } from 'react';

const CORNERS = [
  { style: { bottom: 0, left: '32px' },  hiddenTransform: 'translateY(100%) rotate(-8deg)',          peekTransform: 'translateY(42%) rotate(-8deg)' },
  { style: { bottom: 0, right: '32px' }, hiddenTransform: 'translateY(100%) rotate(8deg)',            peekTransform: 'translateY(42%) rotate(8deg)' },
  { style: { top: 0, left: '32px' },     hiddenTransform: 'translateY(-100%) rotate(8deg) scaleY(-1)', peekTransform: 'translateY(-42%) rotate(8deg) scaleY(-1)' },
  { style: { top: 0, right: '32px' },    hiddenTransform: 'translateY(-100%) rotate(-8deg) scaleY(-1)', peekTransform: 'translateY(-42%) rotate(-8deg) scaleY(-1)' },
];

const INACTIVITY_MS = 10000;

const flyKeyframes = `
@keyframes flyAcross {
  from { transform: translateX(-160px) rotate(-10deg) scaleX(1); }
  to   { transform: translateX(calc(100vw + 160px)) rotate(-10deg) scaleX(1); }
}
@keyframes flyAcrossReverse {
  from { transform: translateX(calc(100vw + 160px)) rotate(10deg) scaleX(-1); }
  to   { transform: translateX(-160px) rotate(10deg) scaleX(-1); }
}
`;

export default function ToucanPeek() {
  const [mode, setMode] = useState(null); // null | 'peek' | 'fly'
  const [corner, setCorner] = useState(0);
  const [flyStyle, setFlyStyle] = useState({});
  const timerRef = useRef(null);
  const modeRef = useRef(null);

  const dismiss = useCallback(() => {
    if (!modeRef.current) return;
    setMode(null);
    modeRef.current = null;
  }, []);

  const startTimer = useCallback(() => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      if (Math.random() < 0.4) {
        // fly across
        const fromLeft = Math.random() < 0.5;
        const topPct = 15 + Math.random() * 55;
        const duration = 2.5 + Math.random() * 1.5;
        setFlyStyle({
          top: `${topPct}%`,
          animation: `${fromLeft ? 'flyAcross' : 'flyAcrossReverse'} ${duration}s linear forwards`,
        });
        modeRef.current = 'fly';
        setMode('fly');
        setTimeout(() => {
          setMode(null);
          modeRef.current = null;
        }, (duration + 0.2) * 1000);
      } else {
        // peek from corner
        setCorner(Math.floor(Math.random() * CORNERS.length));
        modeRef.current = 'peek';
        setMode('peek');
      }
    }, INACTIVITY_MS);
  }, []);

  const handleActivity = useCallback(() => {
    if (modeRef.current === 'peek') dismiss();
    startTimer();
  }, [dismiss, startTimer]);

  useEffect(() => {
    startTimer();
    window.addEventListener('mousemove', handleActivity);
    window.addEventListener('keydown', handleActivity);
    window.addEventListener('click', handleActivity);
    return () => {
      clearTimeout(timerRef.current);
      window.removeEventListener('mousemove', handleActivity);
      window.removeEventListener('keydown', handleActivity);
      window.removeEventListener('click', handleActivity);
    };
  }, [handleActivity, startTimer]);

  const cfg = CORNERS[corner];

  return (
    <>
      <style>{flyKeyframes}</style>

      {/* Peek from corner */}
      <div
        onClick={dismiss}
        style={{
          position: 'fixed',
          zIndex: 9999,
          cursor: 'pointer',
          transition: 'transform 0.6s cubic-bezier(0.34, 1.56, 0.64, 1)',
          transform: mode === 'peek' ? cfg.peekTransform : cfg.hiddenTransform,
          ...cfg.style,
        }}
        title="Shoo!"
      >
        <img
          src={`${import.meta.env.BASE_URL}logos/toucanlogo.png`}
          alt="Toucan"
          style={{ width: '88px', height: '88px', display: 'block', filter: 'drop-shadow(0 4px 12px rgba(0,0,0,0.25))' }}
          draggable={false}
        />
      </div>

      {/* Fly across */}
      {mode === 'fly' && (
        <img
          src={`${import.meta.env.BASE_URL}logos/toucanlogo.png`}
          alt="Toucan"
          style={{
            position: 'fixed',
            zIndex: 9999,
            width: '96px',
            height: '96px',
            filter: 'drop-shadow(0 4px 16px rgba(0,0,0,0.3))',
            pointerEvents: 'none',
            ...flyStyle,
          }}
          draggable={false}
        />
      )}
    </>
  );
}
