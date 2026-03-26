import React from 'react';

export default function LoadingScreen() {
  return (
    <div className="bg-tropical loading-screen">
      <div className="absolute inset-0 pointer-events-none">
        <div className="wave wave1"/>
        <div className="wave wave2"/>
        <div className="wave wave3"/>
      </div>

      {/* Flying toucan */}
      <div className="toucan-fly-outer">
        <div className="toucan-fly-inner">
          <img
            src={`${import.meta.env.BASE_URL}logos/toucanlogo.png`}
            alt="Toucan"
            className="toucan-fly-img"
            draggable={false}
          />
        </div>
      </div>

      {/* Center card */}
      <div className="relative z-10 flex flex-col items-center gap-3">
        <h1 className="loading-title">Toucan</h1>
        <p className="loading-sub">scanning your applications...</p>
        <div className="loading-dots">
          <span/><span/><span/>
        </div>
      </div>
    </div>
  );
}
