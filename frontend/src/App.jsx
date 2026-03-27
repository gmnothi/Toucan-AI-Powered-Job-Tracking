import React, { useState, useEffect, useMemo } from 'react';
import { getJobs, deleteJob, updateJobStatus, triggerRefresh, getStatus, togglePause, getMe, logout } from './api';
import './App.css';
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';
import EmailModal from './components/EmailModal';
import Terminal from './components/Terminal';
import ToucanPeek from './components/ToucanPeek';
import LoadingScreen from './components/LoadingScreen';
import LoginPage from './components/LoginPage';

const STATUS_CONFIG = {
  applied:   { label: 'Applied',      className: 'status-applied',   col: 'col-applied' },
  interview: { label: 'Interviewing', className: 'status-interview', col: 'col-interview' },
  offer:     { label: 'Offer',        className: 'status-offer',     col: 'col-offer' },
  rejected:  { label: 'Rejected',     className: 'status-rejected',  col: 'col-rejected' },
};

const getStatusKey = (status) => {
  const s = (status || '').toLowerCase();
  if (s.startsWith('interview')) return 'interview';
  if (s.startsWith('offer')) return 'offer';
  if (s.startsWith('reject')) return 'rejected';
  return 'applied';
};

const StatusBadge = ({ status }) => {
  const key = getStatusKey(status);
  const cfg = STATUS_CONFIG[key];
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${cfg.className}`}>
      {cfg.label}
    </span>
  );
};

const CompanyLogo = ({ company }) => {
  const formatted = (company || '').toLowerCase().replace(/[^a-z0-9]/g, '').replace(/(inc|llc|corp|corporation|limited|ltd)$/g, '');
  if (!formatted) return null;
  const base = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
  const url = `${base}/logo?domain=${formatted}.com`;
  return (
    <img
      src={url}
      alt=""
      className="w-6 h-6 rounded object-contain flex-shrink-0"
      onError={e => { e.target.style.display = 'none'; }}
    />
  );
};

const StatCard = ({ label, count, colorClass }) => (
  <div className="glass-card rounded-xl px-4 py-2.5 flex items-center gap-2.5">
    <span className={`text-xl font-bold ${colorClass}`}>{count}</span>
    <span className="text-sm text-gray-500">{label}</span>
  </div>
);

export default function App() {
  const [authState, setAuthState] = useState('loading');
  const [currentUser, setCurrentUser] = useState(null);
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isKanbanView, setIsKanbanView] = useState(false);
  const [dateSort, setDateSort] = useState('newest');
  const [selectedJob, setSelectedJob] = useState(null);
  const [terminalOpen, setTerminalOpen] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [paused, setPaused] = useState(false);
  const [scanProgress, setScanProgress] = useState(null);
  const [statusFilter, setStatusFilter] = useState('all');

  const fetchJobs = async () => {
    try {
      const [data] = await Promise.all([
        getJobs(),
        new Promise(res => setTimeout(res, 2000)),
      ]);
      setJobs(data || []);
      setLoading(false);
    } catch (err) {
      setError('Failed to fetch jobs');
      setLoading(false);
    }
  };

  useEffect(() => {
    getMe()
      .then(data => { setCurrentUser(data); setAuthState('authenticated'); })
      .catch(() => setAuthState('unauthenticated'));
  }, []);

  useEffect(() => {
    if (authState === 'authenticated') fetchJobs();
  }, [authState]);

  const handleRefresh = async () => {
    setRefreshing(true);
    setScanProgress(null);
    try {
      await triggerRefresh();
      const poll = setInterval(async () => {
        try {
          const status = await getStatus();
          setScanProgress(status);
          setPaused(status.paused);
          if (!status.running) {
            clearInterval(poll);
            const data = await getJobs();
            setJobs(data || []);
            setRefreshing(false);
          }
        } catch {
          clearInterval(poll);
          setRefreshing(false);
        }
      }, 1000);
    } catch {
      setRefreshing(false);
    }
  };

  const handleDelete = async (jobId) => {
    if (!window.confirm('Delete this job?')) return;
    try {
      await deleteJob(parseInt(jobId));
      await fetchJobs();
    } catch {
      setError('Failed to delete job');
    }
  };

  const sortJobs = (list) => [...list].sort((a, b) => {
    if (!a.Date) return 1;
    if (!b.Date) return -1;
    const diff = new Date(b.Date) - new Date(a.Date);
    return dateSort === 'newest' ? diff : -diff;
  });

  const onDragEnd = async (result) => {
    const { source, destination, draggableId } = result;
    if (!destination) return;
    if (source.droppableId === destination.droppableId && source.index === destination.index) return;

    const jobId = parseInt(draggableId);
    const destKey = destination.droppableId;
    const newStatus = STATUS_CONFIG[destKey].label;

    setJobs(prev => prev.map(job => job.ID === jobId ? { ...job, Status: newStatus } : job));
    try {
      await updateJobStatus(jobId, newStatus);
    } catch {
      await fetchJobs();
    }
  };

  const counts = {
    applied:   jobs.filter(j => getStatusKey(j.Status) === 'applied').length,
    interview: jobs.filter(j => getStatusKey(j.Status) === 'interview').length,
    offer:     jobs.filter(j => getStatusKey(j.Status) === 'offer').length,
    rejected:  jobs.filter(j => getStatusKey(j.Status) === 'rejected').length,
  };

  const kanbanColumns = ['applied', 'interview', 'offer', 'rejected'];
  const jobsByStatus = useMemo(() => kanbanColumns.reduce((acc, key) => {
    acc[key] = sortJobs(jobs.filter(j => getStatusKey(j.Status) === key));
    return acc;
  }, {}), [jobs, dateSort]);

  if (authState === 'loading') return <LoadingScreen />;
  if (authState === 'unauthenticated') return <LoginPage />;
  if (loading) return <LoadingScreen />;

  if (error) return (
    <div className="bg-tropical flex items-center justify-center">
      <div className="absolute inset-0"><div className="wave wave1"/><div className="wave wave2"/><div className="wave wave3"/></div>
      <div className="relative glass-card rounded-xl p-6 text-red-500">{error}</div>
    </div>
  );

  return (
    <div className="bg-tropical">
      <ToucanPeek />
      <EmailModal job={selectedJob} onClose={() => setSelectedJob(null)} />
      <Terminal open={terminalOpen} onClose={() => setTerminalOpen(false)} />

      {/* Animated background */}
      <div className="absolute inset-0 pointer-events-none">
        <div className="wave wave1"/>
        <div className="wave wave2"/>
        <div className="wave wave3"/>
      </div>

      <div className="relative z-10 max-w-7xl mx-auto px-6 py-8">

        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-1">
            <img src={`${import.meta.env.BASE_URL}logos/toucanlogo.png`} className="w-24 h-24 drop-shadow-md" alt="Toucan" />
            <h1 className="text-4xl font-bold text-gray-800 tracking-tight" style={{ fontFamily: 'Inter, sans-serif', letterSpacing: '-0.02em' }}>
              Toucan
            </h1>
          </div>

          <div className="flex items-center gap-3 flex-wrap justify-end">
            <StatCard label="Total"        count={jobs.length}      colorClass="text-gray-500" />
            <StatCard label="Applied"      count={counts.applied}   colorClass="text-indigo-500" />
            <StatCard label="Interviewing" count={counts.interview} colorClass="text-amber-500" />
            <StatCard label="Offers"       count={counts.offer}     colorClass="text-emerald-500" />
            <StatCard label="Rejected"     count={counts.rejected}  colorClass="text-red-400" />

            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="btn-toggle px-4 py-2 rounded-xl text-sm font-semibold flex items-center gap-2 disabled:opacity-50"
            >
              <RefreshIcon spinning={refreshing && !paused} />
              {refreshing ? (paused ? 'Paused' : 'Scanning...') : 'Refresh'}
            </button>

            {refreshing && (
              <button
                onClick={async () => { const r = await togglePause(); setPaused(r.paused); }}
                className="btn-toggle px-4 py-2 rounded-xl text-sm font-semibold flex items-center gap-2"
              >
                {paused ? <PlayIcon /> : <PauseIcon />}
                {paused ? 'Resume' : 'Pause'}
              </button>
            )}

            <button
              onClick={() => setIsKanbanView(!isKanbanView)}
              className="btn-toggle px-4 py-2 rounded-xl text-sm font-semibold flex items-center gap-2 justify-center"
              style={{ minWidth: '100px' }}
            >
              {isKanbanView ? <><TableIcon /> Table</> : <><KanbanIcon /> Kanban</>}
            </button>

            <button
              onClick={() => setTerminalOpen(!terminalOpen)}
              className="btn-toggle px-4 py-2 rounded-xl text-sm font-semibold flex items-center gap-2"
            >
              <TerminalIcon /> Logs
            </button>

            <div className="flex items-center gap-2 pl-2 border-l border-pink-100">
              <span className="text-xs text-gray-400">{currentUser?.email}</span>
              <button
                onClick={async () => { await logout(); setAuthState('unauthenticated'); }}
                className="text-xs text-gray-400 hover:text-red-400 transition-colors"
              >
                Sign out
              </button>
            </div>
          </div>
        </div>

        {/* Progress bar */}
        {refreshing && scanProgress && (
          <div className="glass-card rounded-2xl p-4 mb-6">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-pink-400 animate-pulse" />
                <span className="text-sm font-medium text-gray-700">
                  Scanning {scanProgress.account} inbox
                </span>
              </div>
              <span className="text-sm text-gray-500">
                {scanProgress.processed} / {scanProgress.total} emails &nbsp;·&nbsp;
                <span className="text-emerald-600 font-semibold">{scanProgress.saved} jobs found</span>
              </span>
            </div>
            <div className="w-full bg-pink-100 rounded-full h-2.5 overflow-hidden">
              <div
                className="h-2.5 rounded-full bg-gradient-to-r from-pink-400 to-emerald-400 transition-all duration-300"
                style={{ width: scanProgress.total > 0 ? `${Math.min(100, (scanProgress.processed / scanProgress.total) * 100)}%` : '0%' }}
              />
            </div>
            <p className="text-xs text-gray-400 mt-1.5">
              {scanProgress.total > 0
                ? `${Math.round((scanProgress.processed / scanProgress.total) * 100)}% complete`
                : 'Connecting...'}
            </p>
          </div>
        )}

        {/* View (keyed so switching triggers the enter animation) */}
        <div key={isKanbanView ? 'kanban' : 'table'} className="view-enter">

        {/* Kanban */}
        {isKanbanView ? (
          <DragDropContext onDragEnd={onDragEnd}>
            <div className="grid grid-cols-4 gap-4">
              {kanbanColumns.map(key => {
                const cfg = STATUS_CONFIG[key];
                return (
                  <Droppable droppableId={key} key={key}>
                    {(provided, snapshot) => (
                      <div
                        ref={provided.innerRef}
                        {...provided.droppableProps}
                        className={`glass-card rounded-2xl p-4 min-h-[400px] ${cfg.col} transition-all duration-200 ${snapshot.isDraggingOver ? 'ring-2 ring-pink-300' : ''}`}
                      >
                        <div className="flex items-center justify-between mb-4">
                          <h2 className="text-sm font-semibold text-gray-700">{cfg.label}</h2>
                          <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${cfg.className}`}>
                            {jobsByStatus[key].length}
                          </span>
                        </div>
                        <div className="space-y-2">
                          {jobsByStatus[key].map((job, idx) => (
                            <Draggable draggableId={job.ID.toString()} index={idx} key={job.ID}>
                              {(provided, snapshot) => (
                                <div
                                  ref={provided.innerRef}
                                  {...provided.draggableProps}
                                  {...provided.dragHandleProps}
                                  className={`glass-card rounded-xl p-3 cursor-grab active:cursor-grabbing ${snapshot.isDragging ? 'dragging' : ''}`}
                                >
                                  <div className="flex items-start justify-between gap-2" onClick={() => !snapshot.isDragging && setSelectedJob(job)}>
                                    <div className="min-w-0 flex-1 cursor-pointer">
                                      <p className="text-sm font-semibold text-gray-800 truncate">{job.Title}</p>
                                      <div className="flex items-center gap-1.5 mt-1">
                                        <CompanyLogo company={job.Company} />
                                        <p className="text-xs text-gray-500 truncate">{job.Company}</p>
                                      </div>
                                      {job.Date && <p className="text-xs text-gray-400 mt-1.5">{job.Date}</p>}
                                    </div>
                                    <button
                                      onClick={() => handleDelete(job.ID)}
                                      className="text-red-400 hover:text-red-600 transition-colors flex-shrink-0 p-0.5"
                                    >
                                      <TrashIcon />
                                    </button>
                                  </div>
                                </div>
                              )}
                            </Draggable>
                          ))}
                          {provided.placeholder}
                          {jobsByStatus[key].length === 0 && !snapshot.isDraggingOver && (
                            <p className="text-xs text-gray-300 text-center py-8">Drop here</p>
                          )}
                        </div>
                      </div>
                    )}
                  </Droppable>
                );
              })}
            </div>
          </DragDropContext>
        ) : (
          /* Table */
          <div className="glass-card rounded-2xl overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-pink-100">
                  <th className="text-left px-5 py-4 text-xs font-semibold text-gray-400 uppercase tracking-wider">Company</th>
                  <th className="text-left px-5 py-4 text-xs font-semibold text-gray-400 uppercase tracking-wider">Role</th>
                  <th className="text-left px-5 py-4 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                    <div className="flex items-center gap-2">
                      <span>Status</span>
                      <select
                        value={statusFilter}
                        onChange={e => setStatusFilter(e.target.value)}
                        onClick={e => e.stopPropagation()}
                        className="bg-white/60 border border-sky-200 rounded-lg px-2 py-0.5 text-xs text-gray-500 focus:outline-none focus:ring-2 focus:ring-sky-300 normal-case tracking-normal font-normal"
                      >
                        <option value="all">All</option>
                        <option value="applied">Applied</option>
                        <option value="interview">Interviewing</option>
                        <option value="offer">Offer</option>
                        <option value="rejected">Rejected</option>
                      </select>
                    </div>
                  </th>
                  <th className="text-left px-5 py-4 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                    <div className="flex items-center gap-2">
                      Date
                      <select
                        value={dateSort}
                        onChange={e => setDateSort(e.target.value)}
                        className="bg-white/60 border border-pink-200 rounded-lg px-2 py-0.5 text-xs text-gray-500 focus:outline-none focus:ring-2 focus:ring-pink-300 normal-case tracking-normal font-normal"
                      >
                        <option value="newest">Newest</option>
                        <option value="oldest">Oldest</option>
                      </select>
                    </div>
                  </th>
                  <th className="px-5 py-4" />
                </tr>
              </thead>
              <tbody>
                {sortJobs(jobs.filter(j => statusFilter === 'all' || getStatusKey(j.Status) === statusFilter)).map(job => (
                  <tr key={job.ID} className="table-row cursor-pointer" onClick={() => setSelectedJob(job)}>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-2.5">
                        <CompanyLogo company={job.Company} />
                        <span className="text-sm font-semibold text-gray-700">{job.Company}</span>
                      </div>
                    </td>
                    <td className="px-5 py-3.5 text-sm text-gray-600">{job.Title}</td>
                    <td className="px-5 py-3.5"><StatusBadge status={job.Status} /></td>
                    <td className="px-5 py-3.5 text-sm text-gray-400">{job.Date}</td>
                    <td className="px-5 py-3.5 text-right">
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDelete(job.ID); }}
                        className="text-red-400 hover:text-red-600 transition-colors p-1 rounded-lg"
                      >
                        <TrashIcon />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {jobs.length === 0 && (
              <div className="text-center py-16 text-gray-300">No jobs tracked yet</div>
            )}
          </div>
        )}

        </div>{/* end view-enter */}

        {/* Attribution */}
        <p className="text-center text-xs text-gray-400 mt-6">
          <a href="https://logo.dev" target="_blank" rel="noopener">Logos provided by Logo.dev</a>
        </p>

      </div>
    </div>
  );
}

function PauseIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
    </svg>
  );
}

function PlayIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clipRule="evenodd" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M2 5a2 2 0 012-2h12a2 2 0 012 2v10a2 2 0 01-2 2H4a2 2 0 01-2-2V5zm3.293 1.293a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 01-1.414-1.414L7.586 10 5.293 7.707a1 1 0 010-1.414zM11 12a1 1 0 100 2h3a1 1 0 100-2h-3z" clipRule="evenodd" />
    </svg>
  );
}

function RefreshIcon({ spinning }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className={`h-4 w-4 ${spinning ? 'animate-spin' : ''}`} viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clipRule="evenodd" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
    </svg>
  );
}

function TableIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path fillRule="evenodd" d="M5 4a3 3 0 00-3 3v6a3 3 0 003 3h10a3 3 0 003-3V7a3 3 0 00-3-3H5zm-1 9v-1h5v2H5a1 1 0 01-1-1zm7 1h4a1 1 0 001-1v-1h-5v2zm0-4h5V8h-5v2zM9 8H4v2h5V8z" clipRule="evenodd" />
    </svg>
  );
}

function KanbanIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
      <path d="M2 4a1 1 0 011-1h3a1 1 0 011 1v12a1 1 0 01-1 1H3a1 1 0 01-1-1V4zM8 4a1 1 0 011-1h3a1 1 0 011 1v7a1 1 0 01-1 1H9a1 1 0 01-1-1V4zM14 4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1h-2a1 1 0 01-1-1V4z" />
    </svg>
  );
}
