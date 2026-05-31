import { Spin } from 'antd';
import type { ReactNode } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from './AuthContext';
import type { Role } from '../types/auth';

export function ProtectedRoute({ roles, children }: { roles: Role[]; children: ReactNode }) {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <Spin />;
  }
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  if (!roles.includes(user.role)) {
    return <Navigate to={`/${user.role}`} replace />;
  }
  return children;
}
