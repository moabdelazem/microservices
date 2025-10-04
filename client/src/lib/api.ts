// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost';
const AUTH_SERVICE_URL = `${API_BASE_URL}:3001/api/auth`;
const TASKS_SERVICE_URL = `${API_BASE_URL}:3002/api/tasks`;

export interface User {
  id: string;
  username: string;
  email: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface Task {
  id: string;
  user_id: string;
  title: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed';
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  title: string;
  description: string;
  status?: 'pending' | 'in_progress' | 'completed';
}

export interface UpdateTaskRequest {
  title?: string;
  description?: string;
  status?: 'pending' | 'in_progress' | 'completed';
}

export interface TaskStats {
  total: number;
  pending: number;
  in_progress: number;
  completed: number;
}

// Helper to get auth token from localStorage
const getAuthToken = (): string | null => {
  return localStorage.getItem('token');
};

// Helper to set auth headers
const getAuthHeaders = (): Record<string, string> => {
  const token = getAuthToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
};

// Auth API
export const authAPI = {
  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await fetch(`${AUTH_SERVICE_URL}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Registration failed');
    }

    const result = await response.json();
    return result;
  },

  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(`${AUTH_SERVICE_URL}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    return await response.json();
  },

  async verify(): Promise<User> {
    const response = await fetch(`${AUTH_SERVICE_URL}/verify`, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Token verification failed');
    }

    const result = await response.json();
    return result.user;
  },
};

// Tasks API
export const tasksAPI = {
  async getTasks(status?: string): Promise<Task[]> {
    const url = status
      ? `${TASKS_SERVICE_URL}?status=${status}`
      : TASKS_SERVICE_URL;
    
    const response = await fetch(url, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch tasks');
    }

    const result = await response.json();
    return result.tasks || [];
  },

  async getTask(id: string): Promise<Task> {
    const response = await fetch(`${TASKS_SERVICE_URL}/${id}`, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch task');
    }

    const result = await response.json();
    return result.task;
  },

  async createTask(data: CreateTaskRequest): Promise<Task> {
    const response = await fetch(TASKS_SERVICE_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create task');
    }

    const result = await response.json();
    return result.task;
  },

  async updateTask(id: string, data: UpdateTaskRequest): Promise<Task> {
    const response = await fetch(`${TASKS_SERVICE_URL}/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update task');
    }

    const result = await response.json();
    return result.task;
  },

  async deleteTask(id: string): Promise<void> {
    const response = await fetch(`${TASKS_SERVICE_URL}/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to delete task');
    }
  },

  async getStats(): Promise<TaskStats> {
    const response = await fetch(`${TASKS_SERVICE_URL}/stats/summary`, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch stats');
    }

    const result = await response.json();
    return result.stats;
  },
};
