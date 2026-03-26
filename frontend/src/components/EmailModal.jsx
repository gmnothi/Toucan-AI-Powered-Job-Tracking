import React, { useEffect } from 'react';

export default function EmailModal({ job, onClose }) {
  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onClose]);

  if (!job) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      onClick={onClose}
      style={{ animation: 'fadeIn 0.15s ease' }}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/30 backdrop-blur-sm" />

      {/* Modal */}
      <div
        className="relative glass-card rounded-2xl w-full max-w-2xl max-h-[80vh] flex flex-col shadow-2xl"
        style={{ animation: 'slideUp 0.2s cubic-bezier(0.22,1,0.36,1)' }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-4 p-5 border-b border-pink-100">
          <div className="min-w-0">
            <p className="text-xs text-gray-400 mb-1">{job.Date}</p>
            <h2 className="text-base font-semibold text-gray-800 leading-snug">
              {job.Subject || job.Title}
            </h2>
            <p className="text-sm text-gray-500 mt-0.5">{job.Company}</p>
          </div>
          <button
            onClick={onClose}
            className="flex-shrink-0 text-gray-400 hover:text-gray-600 transition-colors p-1 rounded-lg hover:bg-pink-50"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="overflow-y-auto p-5 flex-1">
          {job.Body ? (
            <pre className="text-sm text-gray-700 whitespace-pre-wrap font-sans leading-relaxed">
              {job.Body}
            </pre>
          ) : (
            <p className="text-sm text-gray-400 italic">No email body stored — rescan to populate.</p>
          )}
        </div>
      </div>
    </div>
  );
}
