import React from 'react';
import { loginWithGoogle } from '../api';

export default function LoginPage() {
  return (
    <div className="bg-tropical min-h-screen flex items-center justify-center">
      <div className="absolute inset-0 pointer-events-none">
        <div className="wave wave1"/>
        <div className="wave wave2"/>
        <div className="wave wave3"/>
      </div>

      <div className="relative z-10 glass-card rounded-2xl p-10 flex flex-col items-center gap-6 max-w-sm w-full mx-4">
        <img
          src="/logos/toucanlogo.png"
          alt="Toucan"
          className="w-24 h-24 drop-shadow-md"
        />
        <div className="text-center">
          <h1 className="text-3xl font-bold text-gray-800 tracking-tight">Toucan</h1>
          <p className="text-gray-500 mt-1 text-sm">Your AI-powered job tracker</p>
        </div>

        <button
          onClick={loginWithGoogle}
          className="w-full flex items-center justify-center gap-3 bg-white border border-gray-200 rounded-xl px-5 py-3 text-sm font-semibold text-gray-700 shadow-sm hover:shadow-md transition-all"
        >
          <GoogleIcon />
          Sign in with Google
        </button>

        <p className="text-xs text-gray-400 text-center">
          We only access your email address — nothing else.
          <br />Your data is private and never shared.
        </p>
      </div>
    </div>
  );
}

function GoogleIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18">
      <path fill="#4285F4" d="M16.51 8H8.98v3h4.3c-.18 1-.74 1.48-1.6 2.04v2.01h2.6a7.8 7.8 0 0 0 2.38-5.88c0-.57-.05-.66-.15-1.18z"/>
      <path fill="#34A853" d="M8.98 17c2.16 0 3.97-.72 5.3-1.94l-2.6-2.04a4.8 4.8 0 0 1-7.18-2.54H1.83v2.07A8 8 0 0 0 8.98 17z"/>
      <path fill="#FBBC05" d="M4.5 10.48A4.8 4.8 0 0 1 4.5 7.5V5.43H1.83a8 8 0 0 0 0 7.12z"/>
      <path fill="#EA4335" d="M8.98 3.58c1.32 0 2.5.45 3.44 1.35l2.58-2.58A8 8 0 0 0 1.83 5.43L4.5 7.5a4.77 4.77 0 0 1 4.48-3.92z"/>
    </svg>
  );
}
