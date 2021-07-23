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

        private static void log(LogLevel logLevel, string logMsg, params object[] modifiers){
            if (level >= logLevel){
                Console.WriteLine("[" + logLevel.ToString().ToUpper() + "] " + logMsg, modifiers);
            }
        }

        public static void SetLevel(LogLevel logLevel){
            level = logLevel;
        }

        public static void Error(string logMsg, params object[] modifiers){
            log(LogLevel.ERROR, logMsg, modifiers);
        }

        public static void Warn(string logMsg, params object[] modifiers)
        {
            log(LogLevel.WARNING, logMsg, modifiers);
        }
        public static void Info(string logMsg, params object[] modifiers)
        {
            log(LogLevel.INFO, logMsg, modifiers);
        }

        public static void Debug(string logMsg, params object[] modifiers)
        {
            log(LogLevel.DEBUG, logMsg, modifiers);
        }

    }
}
