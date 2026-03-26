import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export const getJobs = async () => {
  try {
    const response = await api.get('/jobs');
    return response.data;
  } catch (error) {
    console.error('Error fetching jobs:', error);
    throw error;
  }
};

export const deleteJob = async (jobId) => {
  try {
    await api.delete(`/jobs/${jobId}`);
  } catch (error) {
    console.error('Error deleting job:', error);
    throw error;
  }
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
  try {
    await api.post('/refresh', since ? { since } : {});
  } catch (error) {
    console.error('Error triggering refresh:', error);
    throw error;
  }
};

export const updateJobStatus = async (jobId, status) => {
  try {
    await api.put(`/jobs/${jobId}/status`, { status });
  } catch (error) {
    console.error('Error updating job status:', error);
    throw error;
  }
}; 