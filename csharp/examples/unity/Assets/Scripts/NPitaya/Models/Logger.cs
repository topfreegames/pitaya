using System;

namespace NPitaya.Models{
    public enum LogLevel{
        ERROR=0,
        WARNING=1,
        INFO=2,
        DEBUG=3
    }

    public static class Logger{
        private static LogLevel level = LogLevel.INFO;
        public delegate void LogDelegate(LogLevel level, string logMsg, params object[] modifiers);
        public static LogDelegate LogFunc = new LogDelegate(log);

        private static void log(LogLevel logLevel, string logMsg, params object[] modifiers){
            if (level >= logLevel){
                #if UNITY_STANDALONE || UNITY_EDITOR
                UnityEngine.Debug.Log("[" + logLevel.ToString().ToUpper() + "] " + logMsg);
                #else
                Console.WriteLine("[" + logLevel.ToString().ToUpper() + "] " + logMsg, modifiers);
                #endif
            }
        }

        public static void SetLevel(LogLevel logLevel){
            level = logLevel;
        }

        public static void Error(string logMsg, params object[] modifiers){
            LogFunc(LogLevel.ERROR, logMsg, modifiers);
        }

        public static void Warn(string logMsg, params object[] modifiers)
        {
            LogFunc(LogLevel.WARNING, logMsg, modifiers);
        }
        public static void Info(string logMsg, params object[] modifiers)
        {
            LogFunc(LogLevel.INFO, logMsg, modifiers);
        }

        public static void Debug(string logMsg, params object[] modifiers)
        {
            LogFunc(LogLevel.DEBUG, logMsg, modifiers);
        }

    }
}
