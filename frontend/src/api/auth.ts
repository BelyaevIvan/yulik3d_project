import { request } from './client';
import type { UserDTO } from './types';

export interface RegisterRequest {
  email: string;
  password: string;
  full_name: string;
  phone?: string;
  captcha_token?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface UpdateMeRequest {
  full_name?: string;
  phone?: string;
  old_password?: string;
  new_password?: string;
}

export const authApi = {
  register: (req: RegisterRequest) => request<UserDTO>('/auth/register', { method: 'POST', body: req }),
  login: (req: LoginRequest) => request<UserDTO>('/auth/login', { method: 'POST', body: req }),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  me: () => request<UserDTO>('/me'),
  updateMe: (req: UpdateMeRequest) => request<UserDTO>('/me', { method: 'PATCH', body: req }),

  passwordResetRequest: (email: string, captchaToken: string) =>
    request<{ ok: boolean }>('/auth/password/reset-request', {
      method: 'POST',
      body: { email, captcha_token: captchaToken },
    }),
  passwordResetConfirm: (token: string, newPassword: string) =>
    request<{ ok: boolean }>('/auth/password/reset-confirm', {
      method: 'POST',
      body: { token, new_password: newPassword },
    }),

  emailVerifyConfirm: (token: string) =>
    request<{ ok: boolean }>('/auth/email/verify', { method: 'POST', body: { token } }),
  emailVerifyResend: (email: string, captchaToken: string) =>
    request<{ ok: boolean }>('/auth/email/verify/resend', {
      method: 'POST',
      body: { email, captcha_token: captchaToken },
    }),
};
