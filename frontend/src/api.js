import axios from 'axios';

const BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
const AUTH_BASE = BASE.replace('/api', '');

const TOKEN_KEY = 'toucan_token';

export const saveToken = (token) => localStorage.setItem(TOKEN_KEY, token);
export const clearToken = () => localStorage.removeItem(TOKEN_KEY);
const getToken = () => localStorage.getItem(TOKEN_KEY);

const api = axios.create({
  baseURL: BASE,
  headers: { 'Content-Type': 'application/json' },
});

api.interceptors.request.use((config) => {
  const token = getToken();
  if (token) config.headers['Authorization'] = `Bearer ${token}`;
  return config;
});

export const getMe = async () => {
  const response = await api.get(`${AUTH_BASE}/auth/me`);
  return response.data;
};

export const logout = async () => {
  await api.post(`${AUTH_BASE}/auth/logout`);
  clearToken();
};

export const loginWithGoogle = () => {
  window.location.href = `${AUTH_BASE}/auth/google`;
};

export const getJobs = async () => {
  const response = await api.get('/jobs');
  return response.data;
};

export const deleteJob = async (jobId) => {
  await api.delete(`/jobs/${jobId}`);
};

export const getStatus = async () => {
  const response = await api.get('/status');
  return response.data;
};

export const togglePause = async () => {
  const response = await api.post('/pause');
  return response.data;
};

export const triggerRefresh = async (since = '') => {
  await api.post('/refresh', since ? { since } : {});
};

export const updateJobStatus = async (jobId, status) => {
  await api.put(`/jobs/${jobId}/status`, { status });
};
