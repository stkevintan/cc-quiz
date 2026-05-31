import axios from 'axios';
import type { HealthResponse } from '../types/health';
import type {
  AttemptRecord,
  ClassRecord,
  ManagedUser,
  QuestionRecord,
  Role,
  StudentReportRecord,
  StudentScoreRecord,
  User,
  WrongAnswerRecord,
} from '../types/auth';

const TOKEN_KEY = 'ccq.authToken';

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 5000,
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export function getStoredToken() {
  return localStorage.getItem(TOKEN_KEY);
}

export function setStoredToken(token: string | null) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export async function getHealth(): Promise<HealthResponse> {
  const response = await apiClient.get<HealthResponse>('/api/health');
  return response.data;
}

export async function login(username: string, password: string): Promise<{ token: string; user: User }> {
  const response = await apiClient.post('/api/auth/login', { username, password });
  return response.data;
}

export async function getMe(): Promise<User> {
  const response = await apiClient.get('/api/auth/me');
  return response.data.user;
}

export async function listUsers(role?: Role): Promise<ManagedUser[]> {
  const response = await apiClient.get('/api/admin/users', { params: role ? { role } : undefined });
  return response.data.users;
}

export async function createUser(payload: {
  username: string;
  password: string;
  displayName: string;
  role: 'teacher' | 'student';
  classIds: number[];
}) {
  const response = await apiClient.post('/api/admin/users', payload);
  return response.data;
}

export async function disableUser(id: number) {
  const response = await apiClient.post(`/api/admin/users/${id}/disable`);
  return response.data;
}

export async function resetPassword(id: number, password: string) {
  const response = await apiClient.post(`/api/admin/users/${id}/reset-password`, { password });
  return response.data;
}

export async function listClasses(): Promise<ClassRecord[]> {
  const response = await apiClient.get('/api/admin/classes');
  return response.data.classes;
}

export async function createClass(payload: { name: string; description: string }) {
  const response = await apiClient.post('/api/admin/classes', payload);
  return response.data;
}

export async function bindClassMember(classId: number, role: 'teacher' | 'student', userId: number) {
  const path = role === 'teacher' ? 'teachers' : 'students';
  const response = await apiClient.post(`/api/admin/classes/${classId}/${path}/${userId}`);
  return response.data;
}

export async function listTeacherStudents(): Promise<StudentReportRecord[]> {
  const response = await apiClient.get('/api/teacher/students');
  return response.data.students;
}

export async function listAdminStudents(): Promise<StudentReportRecord[]> {
  const response = await apiClient.get('/api/admin/students');
  return response.data.students;
}

export async function getAdminStudent(id: number): Promise<StudentReportRecord> {
  const response = await apiClient.get(`/api/admin/students/${id}`);
  return response.data.student;
}

export async function listAdminStudentWrongAnswers(id: number): Promise<WrongAnswerRecord[]> {
  const response = await apiClient.get(`/api/admin/students/${id}/wrong-answers`);
  return response.data.wrongAnswers;
}

export async function resetStudentScore(id: number) {
  const response = await apiClient.post(`/api/admin/students/${id}/reset-score`);
  return response.data;
}

export async function resetStudentAttempts(id: number) {
  const response = await apiClient.post(`/api/admin/students/${id}/reset-attempts`);
  return response.data;
}

export async function getTeacherStudent(id: number): Promise<StudentReportRecord> {
  const response = await apiClient.get(`/api/teacher/students/${id}`);
  return response.data.student;
}

export async function listTeacherStudentWrongAnswers(id: number): Promise<WrongAnswerRecord[]> {
  const response = await apiClient.get(`/api/teacher/students/${id}/wrong-answers`);
  return response.data.wrongAnswers;
}

export async function listQuestions(): Promise<QuestionRecord[]> {
  const response = await apiClient.get('/api/questions');
  return response.data.questions;
}

export async function createQuestion(payload: {
  prompt: string;
  options: string[];
  correctOption: number;
  explanation: string;
}) {
  const response = await apiClient.post('/api/questions', payload);
  return response.data;
}

export async function updateQuestion(id: number, payload: {
  prompt: string;
  options: string[];
  correctOption: number;
  explanation: string;
}) {
  const response = await apiClient.patch(`/api/questions/${id}`, payload);
  return response.data;
}

export async function disableQuestion(id: number) {
  const response = await apiClient.post(`/api/questions/${id}/disable`);
  return response.data;
}

export async function startStudentAttempt(): Promise<AttemptRecord> {
  const response = await apiClient.post('/api/student/attempts');
  return response.data.attempt;
}

export async function getCurrentStudentAttempt(): Promise<AttemptRecord | null> {
  const response = await apiClient.get('/api/student/attempts/current');
  return response.data.attempt;
}

export async function getStudentAttempt(id: number): Promise<AttemptRecord> {
  const response = await apiClient.get(`/api/student/attempts/${id}`);
  return response.data.attempt;
}

export async function getStudentScore(): Promise<StudentScoreRecord> {
  const response = await apiClient.get('/api/student/score');
  return response.data.score;
}

export async function listStudentWrongAnswers(): Promise<WrongAnswerRecord[]> {
  const response = await apiClient.get('/api/student/wrong-answers');
  return response.data.wrongAnswers;
}

export async function submitStudentAttemptAnswer(
  attemptId: number,
  payload: { attemptQuestionId: number; selectedOption: number },
): Promise<{
  attempt: AttemptRecord;
  answer: { attemptQuestionId: number; selectedOption: number; isCorrect: boolean; correctOption: number; scoreDelta: number };
}> {
  const response = await apiClient.post(`/api/student/attempts/${attemptId}/answers`, payload);
  return response.data;
}
