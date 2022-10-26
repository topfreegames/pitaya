namespace NPitaya.Models
{
    [System.AttributeUsage(System.AttributeTargets.Class | System.AttributeTargets.Struct)]
    public class Remote: System.Attribute
    {
        private string _name;

        public Remote(string name)
        {
            _name = name;
        }

        public Remote()
        {
            _name = Utils.Utils.DefaultRemoteNameFunc(GetType().Name);
        }
    }
}
