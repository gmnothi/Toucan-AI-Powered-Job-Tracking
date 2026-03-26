import React, { useEffect, useRef, useState } from 'react';

const TYPE_STYLES = {
  flagged: 'text-emerald-500 font-semibold',
  skipped: 'text-gray-400',
  reading: 'text-pink-400',
  info:    'text-indigo-400 font-medium',
  error:   'text-red-400',
};

export default function Terminal({ open, onClose }) {
  const [lines, setLines] = useState([]);
  const bottomRef = useRef(null);
  const esRef = useRef(null);

  useEffect(() => {
    if (!open) return;

    const es = new EventSource('http://localhost:8080/api/logs');
    esRef.current = es;

    es.addEventListener('log', (e) => {
      try {
        const entry = JSON.parse(e.data);
        setLines(prev => [...prev.slice(-500), entry]); // keep last 500 lines
      } catch {}
    });

    return () => es.close();
  }, [open]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  if (!open) return null;

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40" style={{ animation: 'slideUp 0.25s cubic-bezier(0.22,1,0.36,1)' }}>
      <div className="glass-card rounded-t-2xl border-b-0 mx-4 shadow-2xl flex flex-col" style={{ height: '280px' }}>
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-2.5 border-b border-pink-100 flex-shrink-0">
          <div className="flex items-center gap-2">
            <div className="flex gap-1.5">
              <div className="w-3 h-3 rounded-full bg-red-400" />
              <div className="w-3 h-3 rounded-full bg-yellow-400" />
              <div className="w-3 h-3 rounded-full bg-green-400" />
            </div>
            <span className="text-xs font-mono font-semibold text-gray-500 ml-1">email processor</span>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={() => setLines([])}
              className="text-xs text-gray-400 hover:text-gray-600 transition-colors"
            >
              clear
            </button>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
              </svg>
            </button>
          </div>
        </div>

        {/* Log output */}
        <div className="flex-1 overflow-y-auto px-4 py-3 font-mono text-xs">
          {lines.length === 0 ? (
            <p className="text-gray-300 italic">Waiting for scan activity...</p>
          ) : (
            lines.map((line, i) => (
              <div key={i} className={`leading-5 ${TYPE_STYLES[line.type] || 'text-gray-500'}`}>
                {line.message}
              </div>
            ))
          )}
          <div ref={bottomRef} />
        </div>
      </div>
    </div>
  );
}
