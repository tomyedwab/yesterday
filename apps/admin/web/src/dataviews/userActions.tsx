import { useState } from 'react';
import { useConnectionDispatch, CreatePendingEvent } from '@tomyedwab/yesterday';

export interface UserActionResult {
  success: boolean;
  error?: string;
}

export interface UpdateUserRequest {
  userID: number;
  username: string;
}

export interface UpdatePasswordRequest {
  userID: number;
  newPassword: string;
}

export interface DeleteUserRequest {
  userID: number;
}

export const useUpdateUser = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const updateUser = async (request: UpdateUserRequest): Promise<UserActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("updateuser:", {
          type: "UpdateUser",
          user_id: request.userID,
          username: request.username,
        })
      );
      return { success: true };
    } catch (error) {
      return { success: false, error: error instanceof Error ? error.message : 'Unknown error' };
    } finally {
      setIsLoading(false);
    }
  };

  return { updateUser, isLoading };
};

export const useUpdatePassword = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const updatePassword = async (request: UpdatePasswordRequest): Promise<UserActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("updatepassword:", {
          type: "UpdateUserPassword",
          user_id: request.userID,
          new_password: request.newPassword,
        })
      );
      return { success: true };
    } catch (error) {
      return { success: false, error: error instanceof Error ? error.message : 'Unknown error' };
    } finally {
      setIsLoading(false);
    }
  };

  return { updatePassword, isLoading };
};

export const useDeleteUser = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const deleteUser = async (request: DeleteUserRequest): Promise<UserActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("deleteuser:", {
          type: "DeleteUser",
          user_id: request.userID,
        })
      );
      return { success: true };
    } catch (error) {
      return { success: false, error: error instanceof Error ? error.message : 'Unknown error' };
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteUser, isLoading };
};