using System;
using System.Threading.Tasks;
using ExampleORM.Models;
using Gen.Protos;
using Microsoft.EntityFrameworkCore;
using NPitaya.Models;
using User = Gen.Protos.User;

namespace ExampleORM.Servers.BusinessLogic.Handlers
{
  public class UserHandler : BaseHandler
  {
    private async Task<Models.User> GetUserFromToken(Guid token, ExampleContext ctx)
    {
      return await ctx.Users.SingleAsync(u => u.Token == token);
    }

    private User ModelsUserToProtosUser(Models.User modelsUser)
    {
      return new User
      {
        Id = modelsUser.Id.ToString(),
        Name = modelsUser.Name,
        Token = modelsUser.Token.ToString()
      };
    }

    public async Task<User> Authenticate(PitayaSession session, AuthenticateArgs args)
    {
      using (var context = new ExampleContext())
      {
        if (args.Token.Length > 0)
        {
          return ModelsUserToProtosUser(await GetUserFromToken(new Guid(args.Token), context));
        }

        // if token was not sent, create an user and return!
        var newUser = new Models.User();
        newUser = context.Users.Add(newUser).Entity;
        await context.SaveChangesAsync();
        return ModelsUserToProtosUser(newUser);
      }
    }

    public async Task<Answer> ChangeName(PitayaSession session, ChangeNameArgs arg)
    {
      using (var context = new ExampleContext())
      {
        var userModel = await GetUserFromToken(new Guid(arg.Token), context);
        userModel.Name = arg.Name;
        context.Update(userModel);
        await context.SaveChangesAsync();
        return new Answer
        {
          Code = "OK!"
        };
      }
    }
  }
}