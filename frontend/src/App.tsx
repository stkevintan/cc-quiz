import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AuthProvider } from './auth/AuthContext';
import { ProtectedRoute } from './auth/ProtectedRoute';
import { AppLayout } from './layouts/AppLayout';
import { AdminClassesPage } from './pages/admin/AdminClassesPage';
import { AdminDashboard } from './pages/admin/AdminDashboard';
import { AdminUsersPage } from './pages/admin/AdminUsersPage';
import { AdminStudentReportsPage } from './pages/admin/AdminStudentReportsPage';
import { LoginPage } from './pages/LoginPage';
import { QuestionManagementPage } from './pages/questions/QuestionManagementPage';
import { StudentAttemptPage } from './pages/student/StudentAttemptPage';
import { StudentDashboard } from './pages/student/StudentDashboard';
import { TeacherStudentsPage } from './pages/teacher/TeacherStudentsPage';

export default function App() {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#7c3aed',
          borderRadius: 8,
        },
      }}
    >
      <BrowserRouter>
        <AuthProvider>
          <AppLayout>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/" element={<Navigate to="/login" replace />} />
              <Route
                path="/admin"
                element={<ProtectedRoute roles={['admin']}><AdminDashboard /></ProtectedRoute>}
              />
              <Route
                path="/admin/users"
                element={<ProtectedRoute roles={['admin']}><AdminUsersPage /></ProtectedRoute>}
              />
              <Route
                path="/admin/classes"
                element={<ProtectedRoute roles={['admin']}><AdminClassesPage /></ProtectedRoute>}
              />
              <Route
                path="/admin/questions"
                element={<ProtectedRoute roles={['admin']}><QuestionManagementPage /></ProtectedRoute>}
              />
              <Route
                path="/admin/reports"
                element={<ProtectedRoute roles={['admin']}><AdminStudentReportsPage /></ProtectedRoute>}
              />
              <Route
                path="/teacher"
                element={<ProtectedRoute roles={['teacher']}><TeacherStudentsPage /></ProtectedRoute>}
              />
              <Route
                path="/teacher/students"
                element={<ProtectedRoute roles={['teacher']}><TeacherStudentsPage /></ProtectedRoute>}
              />
              <Route
                path="/teacher/questions"
                element={<ProtectedRoute roles={['teacher']}><QuestionManagementPage /></ProtectedRoute>}
              />
              <Route
                path="/student"
                element={<ProtectedRoute roles={['student']}><StudentDashboard /></ProtectedRoute>}
              />
              <Route
                path="/student/attempts/:attemptId"
                element={<ProtectedRoute roles={['student']}><StudentAttemptPage /></ProtectedRoute>}
              />
              <Route path="*" element={<Navigate to="/login" replace />} />
            </Routes>
          </AppLayout>
        </AuthProvider>
      </BrowserRouter>
    </ConfigProvider>
  );
}
