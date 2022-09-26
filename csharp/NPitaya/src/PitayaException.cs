using System;

namespace NPitaya
{
    public class PitayaException : Exception
    {
        public PitayaException()
            : base() { }

        public PitayaException(string message)
            : base(message) { }

        public PitayaException(string format, params object[] args)
            : base(string.Format(format, args)) { }

        public PitayaException(string message, Exception innerException)
            : base(message, innerException) { }

        public PitayaException(string format, Exception innerException, params object[] args)
            : base(string.Format(format, args), innerException) { }
    }
}