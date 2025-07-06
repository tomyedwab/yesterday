import { useState } from "react";
import {
  useConnectionDispatch,
  CreatePendingEvent,
} from "@tomyedwab/yesterday";

export interface ApplicationActionResult {
  success: boolean;
  error?: string;
}

export interface CreateApplicationRequest {
  appId: string;
  displayName: string;
  hostName: string;
}

export interface CreateDebugApplicationRequest {
  appId: string;
  displayName: string;
  hostName: string;
  staticServiceUrl?: string;
}

export interface UpdateApplicationRequest {
  instanceId: string;
  appId: string;
  displayName: string;
  hostName: string;
}

export interface DeleteApplicationRequest {
  instanceId: string;
}

export const useCreateApplication = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const createApplication = async (
    request: CreateApplicationRequest,
  ): Promise<ApplicationActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("createapp:", {
          type: "CreateApplication",
          appId: request.appId,
          displayName: request.displayName,
          hostName: request.hostName,
        }),
      );
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    } finally {
      setIsLoading(false);
    }
  };

  return { createApplication, isLoading };
};

export const useCreateDebugApplication = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const createDebugApplication = async (
    request: CreateDebugApplicationRequest,
  ): Promise<ApplicationActionResult> => {
    setIsLoading(true);
    try {
      const payload: any = {
        type: "CreateDebugApplication",
        appId: request.appId,
        displayName: request.displayName,
        hostName: request.hostName,
      };
      
      if (request.staticServiceUrl) {
        payload.staticServiceUrl = request.staticServiceUrl;
      }
      
      connectDispatch(CreatePendingEvent("createdebugapp:", payload));
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    } finally {
      setIsLoading(false);
    }
  };

  return { createDebugApplication, isLoading };
};

export const useUpdateApplication = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const updateApplication = async (
    request: UpdateApplicationRequest,
  ): Promise<ApplicationActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("updateapp:", {
          type: "UpdateApplication",
          instanceId: request.instanceId,
          appId: request.appId,
          displayName: request.displayName,
          hostName: request.hostName,
        }),
      );
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    } finally {
      setIsLoading(false);
    }
  };

  return { updateApplication, isLoading };
};

export const useDeleteApplication = () => {
  const [isLoading, setIsLoading] = useState(false);
  const connectDispatch = useConnectionDispatch();

  const deleteApplication = async (
    request: DeleteApplicationRequest,
  ): Promise<ApplicationActionResult> => {
    setIsLoading(true);
    try {
      connectDispatch(
        CreatePendingEvent("deleteapp:", {
          type: "DeleteApplication",
          instanceId: request.instanceId,
        }),
      );
      return { success: true };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
      };
    } finally {
      setIsLoading(false);
    }
  };

  return { deleteApplication, isLoading };
};
