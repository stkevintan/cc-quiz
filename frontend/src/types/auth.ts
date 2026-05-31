export type Role = 'admin' | 'teacher' | 'student';

export interface User {
  id: number;
  username: string;
  displayName: string;
  role: Role;
  status: 'active' | 'disabled';
}

export interface ClassSummary {
  id: number;
  name: string;
}

export interface ManagedUser extends User {
  classes?: ClassSummary[];
}

export interface ClassRecord {
  id: number;
  name: string;
  description: string;
  status: string;
  teachers: ManagedUser[];
  students: ManagedUser[];
}

export interface QuestionRecord {
  id: number;
  prompt: string;
  options: string[];
  optionCount: number;
  correctOption: number;
  explanation: string;
  status: 'active' | 'disabled';
  createdBy?: number;
  createdAt?: string;
  updatedAt?: string;
}

export interface AttemptAnswerRecord {
  id: number;
  attemptId: number;
  attemptQuestionId: number;
  selectedOption: number;
  isCorrect: boolean;
  scoreDelta: number;
  answeredAt: string;
}

export interface AttemptQuestionRecord {
  id: number;
  attemptId: number;
  questionId: number;
  position: number;
  prompt: string;
  options: string[];
  optionCount: number;
  correctOption?: number;
  explanation?: string;
  answer?: AttemptAnswerRecord;
}

export interface AttemptRecord {
  id: number;
  studentId: number;
  status: 'in_progress' | 'completed';
  questionCount: number;
  answeredCount: number;
  scoreDelta: number;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  questions: AttemptQuestionRecord[];
}

export interface StudentScoreRecord {
  studentId: number;
  totalScore: number;
  updatedAt?: string;
}

export interface WrongAnswerRecord {
  id: number;
  studentId: number;
  attemptId: number;
  questionId: number;
  prompt: string;
  options: string[];
  optionCount: number;
  selectedOption: number;
  correctOption: number;
  explanation: string;
  updatedAt: string;
}


export interface AttemptSummaryRecord {
  id: number;
  studentId: number;
  status: 'in_progress' | 'completed';
  questionCount: number;
  answeredCount: number;
  scoreDelta: number;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface StudentReportRecord extends ManagedUser {
  score: StudentScoreRecord;
  attemptCount: number;
  completedCount: number;
  inProgress: boolean;
  wrongCount: number;
  attempts?: AttemptSummaryRecord[];
}
