import React, { useState, useEffect, useRef, useCallback } from 'react';

const CORNERS = [
  // bottom-left
  {
    style: { bottom: 0, left: '32px' },
    hiddenTransform: 'translateY(100%) rotate(-8deg)',
    peekTransform:   'translateY(42%) rotate(-8deg)',
  },
  // bottom-right
  {
    style: { bottom: 0, right: '32px' },
    hiddenTransform: 'translateY(100%) rotate(8deg)',
    peekTransform:   'translateY(42%) rotate(8deg)',
  },
  // top-left (flipped upside-down)
  {
    style: { top: 0, left: '32px' },
    hiddenTransform: 'translateY(-100%) rotate(8deg) scaleY(-1)',
    peekTransform:   'translateY(-42%) rotate(8deg) scaleY(-1)',
  },
  // top-right (flipped upside-down)
  {
    style: { top: 0, right: '32px' },
    hiddenTransform: 'translateY(-100%) rotate(-8deg) scaleY(-1)',
    peekTransform:   'translateY(-42%) rotate(-8deg) scaleY(-1)',
  },
];

const INACTIVITY_MS = 10000;

export default function ToucanPeek() {
  const [visible, setVisible] = useState(false);
  const [corner, setCorner] = useState(0);
  const timerRef = useRef(null);
  const visibleRef = useRef(false);

  const dismiss = useCallback(() => {
    if (!visibleRef.current) return;
    setVisible(false);
    visibleRef.current = false;
  }, []);

  const startTimer = useCallback(() => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      const c = Math.floor(Math.random() * CORNERS.length);
      setCorner(c);
      setVisible(true);
      visibleRef.current = true;
    }, INACTIVITY_MS);
  }, []);

  const handleActivity = useCallback(() => {
    if (visibleRef.current) {
      dismiss();
    }
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
    <div
      onClick={dismiss}
      style={{
        position: 'fixed',
        zIndex: 9999,
        cursor: 'pointer',
        transition: 'transform 0.6s cubic-bezier(0.34, 1.56, 0.64, 1)',
        transform: visible ? cfg.peekTransform : cfg.hiddenTransform,
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
  );
}
