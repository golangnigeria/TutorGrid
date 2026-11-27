import { API_URL } from "@/config/env";
import { apiContract } from "@boilerplate/openapi/contracts";
import { initClient } from "@ts-rest/core";
import axios, {
  type Method,
  AxiosError,
  isAxiosError,
  type AxiosResponse,
} from "axios";

type Headers = Awaited<
  ReturnType<NonNullable<Parameters<typeof initClient>[1]["api"]>>
>["headers"];

export type TApiClient = ReturnType<typeof useApiClient>;

export const useApiClient = ({ isBlob = false }: { isBlob?: boolean } = {}) => {
  return initClient(apiContract, {
    baseUrl: "",
    baseHeaders: {
      "Content-Type": "application/json",
    },
    api: async ({ path, method, headers, body }) => {
      try {
        const result = await axios.request({
          method: method as Method,
          url: `${API_URL}/api${path}`,
          withCredentials: true,
          headers: {
            ...headers,
          },
          data: body,
          ...(isBlob ? { responseType: "blob" } : {}),
        });
        return {
          status: result.status,
          body: result.data,
          headers: result.headers as unknown as Headers,
        };
      } catch (e: unknown) {
        if (isAxiosError(e)) {
          const error = e as AxiosError;
          const response = error.response as AxiosResponse | undefined;
          return {
            status: response?.status || 500,
            body: response?.data || { message: "Internal server error" },
            headers: (response?.headers as unknown as Headers) || {},
          };
        }
        throw e;
      }
    },
  });
};
